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

	err := serveHTTP(context.Background(), discardLogger(), server, time.Second, nil, nil)
	if !errors.Is(err, want) {
		t.Fatalf("serveHTTP() error = %v, want wrapped %v", err, want)
	}
}

func TestServeHTTPDrainsThenJoinsServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	routingWaitStarted := make(chan struct{})
	continueRouting := make(chan struct{})
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
		result <- serveHTTP(
			ctx,
			discardLogger(),
			server,
			time.Second,
			func() { ready.Store(false) },
			func(context.Context) error {
				close(routingWaitStarted)
				<-continueRouting
				return nil
			},
		)
	}()
	cancel()
	awaitSignal(t, routingWaitStarted, "routing propagation wait was not started")
	if ready.Load() {
		t.Fatal("server was still ready while waiting for routing propagation")
	}
	select {
	case <-shutdownCalled:
		t.Fatal("Shutdown was called before routing propagation completed")
	default:
	}
	close(continueRouting)
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

	err := serveHTTP(ctx, discardLogger(), server, time.Millisecond, nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("serveHTTP() error = %v, want context.DeadlineExceeded", err)
	}
	select {
	case <-closed:
	default:
		t.Fatal("server was not force closed after the shutdown deadline")
	}
}

func TestServeHTTPForceClosesWhenRoutingWaitExceedsDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	serveDone := make(chan struct{})
	closed := make(chan struct{})
	server := &fakeServer{
		serveDone: serveDone,
		serveErr:  http.ErrServerClosed,
		close: func() error {
			close(serveDone)
			close(closed)
			return nil
		},
	}

	err := serveHTTP(ctx, discardLogger(), server, time.Millisecond, nil, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("serveHTTP() error = %v, want context.DeadlineExceeded", err)
	}
	select {
	case <-closed:
	default:
		t.Fatal("server was not force closed after the routing wait exceeded the deadline")
	}
}

func TestDurationFromEnv(t *testing.T) {
	const name = "TASKAPI_TEST_DURATION"
	t.Run("fallback", func(t *testing.T) {
		t.Setenv(name, "")
		got, err := durationFromEnv(name, 3*time.Second)
		if err != nil || got != 3*time.Second {
			t.Fatalf("durationFromEnv() = %v, %v; want 3s, nil", got, err)
		}
	})
	t.Run("configured", func(t *testing.T) {
		t.Setenv(name, "750ms")
		got, err := durationFromEnv(name, 0)
		if err != nil || got != 750*time.Millisecond {
			t.Fatalf("durationFromEnv() = %v, %v; want 750ms, nil", got, err)
		}
	})
	for _, value := range []string{"later", "-1s"} {
		t.Run("reject_"+value, func(t *testing.T) {
			t.Setenv(name, value)
			if _, err := durationFromEnv(name, 0); err == nil {
				t.Fatalf("durationFromEnv() accepted %q", value)
			}
		})
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
