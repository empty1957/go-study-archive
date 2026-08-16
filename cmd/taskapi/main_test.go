package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestServeHTTPReturnsUnexpectedServeError(t *testing.T) {
	want := errors.New("listen failed")
	server := &fakeServer{serveDone: closedChannel(), serveErr: want}

	err := serveHTTP(context.Background(), discardLogger(), server, time.Second, nil)
	if !errors.Is(err, want) {
		t.Fatalf("serveHTTP() error = %v, want wrapped %v", err, want)
	}
}

func TestServeHTTPDrainsThenJoinsServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	shutdownCalled := make(chan struct{})
	var ready atomic.Bool
	ready.Store(true)
	server := &fakeServer{
		serveDone: serveDone,
		serveErr:  http.ErrServerClosed,
		shutdown: func(context.Context) error {
			if ready.Load() {
				t.Error("server was still ready when shutdown started")
			}
			close(shutdownCalled)
			return nil
		},
	}

	result := make(chan error, 1)
	go func() {
		result <- serveHTTP(ctx, discardLogger(), server, time.Second, func() {
			ready.Store(false)
		})
	}()
	cancel()
	awaitSignal(t, shutdownCalled, "Shutdown was not called")

	select {
	case err := <-result:
		t.Fatalf("serveHTTP() returned before the server goroutine stopped: %v", err)
	default:
	}
	close(serveDone)
	if err := awaitResult(t, result, "serveHTTP did not join the server goroutine"); err != nil {
		t.Fatalf("serveHTTP() error = %v, want nil", err)
	}
}

func TestServeHTTPForceClosesAfterShutdownDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	serveDone := make(chan struct{})
	closed := make(chan struct{})
	server := &fakeServer{
		serveDone: serveDone,
		serveErr:  http.ErrServerClosed,
		shutdown: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		close: func() error {
			close(serveDone)
			close(closed)
			return nil
		},
	}

	err := serveHTTP(ctx, discardLogger(), server, time.Millisecond, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("serveHTTP() error = %v, want context.DeadlineExceeded", err)
	}
	select {
	case <-closed:
	default:
		t.Fatal("server was not force closed after the shutdown deadline")
	}
}

type fakeServer struct {
	serveDone chan struct{}
	serveErr  error
	shutdown  func(context.Context) error
	close     func() error
}

func (s *fakeServer) ListenAndServe() error {
	<-s.serveDone
	return s.serveErr
}

func (s *fakeServer) Shutdown(ctx context.Context) error {
	if s.shutdown != nil {
		return s.shutdown(ctx)
	}
	return nil
}

func (s *fakeServer) Close() error {
	if s.close != nil {
		return s.close()
	}
	return nil
}

func closedChannel() chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

func awaitSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}

func awaitResult(t *testing.T, result <-chan error, message string) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatal(message)
		return nil
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
