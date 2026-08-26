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

const (
	routingDrainDelayEnv = "TASKAPI_ROUTING_DRAIN_DELAY"
	shutdownBudget       = 20 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	routingDrainDelay, err := durationFromEnv(routingDrainDelayEnv, 0)
	if err != nil {
		return err
	}

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
	return serveHTTP(stopCtx, logger, server, shutdownBudget, func() {
		ready.Store(false)
	}, func(ctx context.Context) error {
		if routingDrainDelay == 0 {
			return nil
		}
		logger.Info("waiting for routing propagation", "delay", routingDrainDelay)
		return waitForDuration(ctx, routingDrainDelay)
	})
}

type managedServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Close() error
}

func serveHTTP(
	ctx context.Context,
	logger *slog.Logger,
	server managedServer,
	shutdownTimeout time.Duration,
	beforeShutdown func(),
	waitForRouting func(context.Context) error,
) error {
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

	// The timeout is the application's total shutdown budget. Routing
	// propagation must not silently consume time reserved for in-flight work.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if waitForRouting != nil {
		if err := waitForRouting(shutdownCtx); err != nil {
			return forceClose(server, serveErr, fmt.Errorf("wait for routing propagation: %w", err))
		}
	}

	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		return forceClose(server, serveErr, fmt.Errorf("graceful shutdown: %w", shutdownErr))
	}

	// CancelFunc only broadcasts a signal. Receiving the server result is the
	// join that proves the owner goroutine has actually stopped.
	if err := normalizeServeError(<-serveErr); err != nil {
		return err
	}
	logger.Info("shutdown complete")
	return nil
}

func durationFromEnv(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration < 0 {
		return 0, fmt.Errorf("parse %s: duration must not be negative", name)
	}
	return duration, nil
}

func waitForDuration(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func forceClose(server managedServer, serveErr <-chan error, cause error) error {
	return errors.Join(
		cause,
		wrapError("force close server", server.Close()),
		normalizeServeError(<-serveErr),
	)
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
