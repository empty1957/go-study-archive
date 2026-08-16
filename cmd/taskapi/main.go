package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"example.com/go-cloud-native-study/internal/task"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	store := &task.MemoryStore{}
	service := task.NewService(store)
	var ready atomic.Bool
	ready.Store(true)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           task.NewHandlerWithReadiness(service, logger, ready.Load),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Info("server listening", "address", server.Addr)
	return serveHTTP(stopCtx, logger, server, 10*time.Second, func() {
		ready.Store(false)
	})
}

type managedServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Close() error
}

func serveHTTP(ctx context.Context, logger *slog.Logger, server managedServer, shutdownTimeout time.Duration, beforeShutdown func()) error {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return normalizeServeError(err)
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	if beforeShutdown != nil {
		beforeShutdown()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		closeErr := server.Close()
		return errors.Join(
			fmt.Errorf("graceful shutdown: %w", shutdownErr),
			wrapError("force close server", closeErr),
			normalizeServeError(<-serveErr),
		)
	}

	// CancelFunc only broadcasts a signal. Receiving the server result is the
	// join that proves the owner goroutine has actually stopped.
	return normalizeServeError(<-serveErr)
}

func normalizeServeError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("serve HTTP: %w", err)
}

func wrapError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
