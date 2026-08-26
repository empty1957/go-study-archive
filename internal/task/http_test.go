package task

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandlerCreateAndGet(t *testing.T) {
	handler := newTestHandler()

	create := httptest.NewRequest(http.MethodPost, "/v1/tasks", strings.NewReader(`{"title":"learn Go"}`))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)

	if created.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d; body=%s", created.Code, http.StatusCreated, created.Body.String())
	}
	location := created.Header().Get("Location")
	if location == "" {
		t.Fatal("POST response must include Location")
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, location, nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", get.Code, http.StatusOK)
	}
	var body taskResponse
	if err := json.NewDecoder(get.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Title != "learn Go" {
		t.Errorf("GET title = %q, want learn Go", body.Title)
	}
}

func TestHandlerRejectsBadRequests(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
	}{
		{name: "unknown field", contentType: "application/json", body: `{"title":"x","admin":true}`, wantStatus: http.StatusBadRequest},
		{name: "multiple values", contentType: "application/json", body: `{"title":"x"} {"title":"y"}`, wantStatus: http.StatusBadRequest},
		{name: "wrong media type", contentType: "text/plain", body: `{"title":"x"}`, wantStatus: http.StatusUnsupportedMediaType},
		{name: "missing media type", body: `{"title":"x"}`, wantStatus: http.StatusUnsupportedMediaType},
		{name: "media type prefix", contentType: "application/json-seq", body: `{"title":"x"}`, wantStatus: http.StatusUnsupportedMediaType},
		{name: "empty title", contentType: "application/json", body: `{"title":" "}`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()
			newTestHandler().ServeHTTP(response, req)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}

func TestHandlerRejectsOversizedBody(t *testing.T) {
	body := `{"title":"` + strings.Repeat("x", maxRequestBody) + `"}`
	tests := []struct {
		name     string
		streamed bool
	}{
		{name: "known length"},
		{name: "streamed", streamed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/tasks", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.streamed {
				req.ContentLength = -1
			}
			response := httptest.NewRecorder()

			newTestHandler().ServeHTTP(response, req)

			if response.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
			}
			var failure errorResponse
			if err := json.NewDecoder(response.Body).Decode(&failure); err != nil {
				t.Fatal(err)
			}
			if failure.Code != "request_too_large" {
				t.Fatalf("error code = %q, want request_too_large", failure.Code)
			}
		})
	}
}

func TestHandlerRejectsOverloadBeforeStartingWork(t *testing.T) {
	store := newBlockingCreateStore()
	t.Cleanup(store.unblock)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandlerWithReadinessAndLimit(NewService(store), logger, func() bool { return true }, 1)

	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		request := httptest.NewRequest(http.MethodPost, "/v1/tasks", strings.NewReader(`{"title":"first"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		firstDone <- response
	}()
	select {
	case <-store.started:
	case <-time.After(time.Second):
		t.Fatal("first request did not reach the store")
	}

	overloaded := httptest.NewRecorder()
	handler.ServeHTTP(overloaded, httptest.NewRequest(http.MethodGet, "/v1/tasks", nil))
	if overloaded.Code != http.StatusServiceUnavailable {
		t.Fatalf("overload status = %d, want %d; body=%s", overloaded.Code, http.StatusServiceUnavailable, overloaded.Body.String())
	}
	if got := overloaded.Header().Get("Retry-After"); got != retryAfterSeconds {
		t.Fatalf("Retry-After = %q, want %q", got, retryAfterSeconds)
	}
	var failure errorResponse
	if err := json.NewDecoder(overloaded.Body).Decode(&failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != "overloaded" {
		t.Fatalf("error code = %q, want overloaded", failure.Code)
	}

	for _, path := range []string{"/healthz", "/readyz"} {
		probe := httptest.NewRecorder()
		handler.ServeHTTP(probe, httptest.NewRequest(http.MethodGet, path, nil))
		if probe.Code != http.StatusOK {
			t.Fatalf("GET %s status during overload = %d, want %d", path, probe.Code, http.StatusOK)
		}
	}

	store.unblock()
	select {
	case response := <-firstDone:
		if response.Code != http.StatusCreated {
			t.Fatalf("first request status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
		}
	case <-time.After(time.Second):
		t.Fatal("first request did not finish after the store was released")
	}
	after := httptest.NewRecorder()
	handler.ServeHTTP(after, httptest.NewRequest(http.MethodGet, "/v1/tasks", nil))
	if after.Code != http.StatusOK {
		t.Fatalf("status after release = %d, want %d; body=%s", after.Code, http.StatusOK, after.Body.String())
	}
}

func TestHandlerNotFound(t *testing.T) {
	response := httptest.NewRecorder()
	newTestHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/tasks/missing", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestHandlerReadinessCanDrainWithoutFailingLiveness(t *testing.T) {
	var ready atomic.Bool
	ready.Store(true)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandlerWithReadiness(NewService(&MemoryStore{}), logger, ready.Load)

	assertStatus := func(path string, want int) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != want {
			t.Fatalf("GET %s status = %d, want %d; body=%s", path, response.Code, want, response.Body.String())
		}
	}

	assertStatus("/readyz", http.StatusOK)
	ready.Store(false)
	assertStatus("/readyz", http.StatusServiceUnavailable)
	assertStatus("/healthz", http.StatusOK)
}

func newTestHandler() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(NewService(&MemoryStore{}), logger)
}

type blockingCreateStore struct {
	MemoryStore
	started     chan struct{}
	release     chan struct{}
	once        sync.Once
	releaseOnce sync.Once
}

func (s *blockingCreateStore) unblock() {
	s.releaseOnce.Do(func() { close(s.release) })
}

func newBlockingCreateStore() *blockingCreateStore {
	return &blockingCreateStore{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (s *blockingCreateStore) Create(ctx context.Context, item Task) error {
	s.once.Do(func() { close(s.started) })
	select {
	case <-s.release:
		return s.MemoryStore.Create(ctx, item)
	case <-ctx.Done():
		return ctx.Err()
	}
}
