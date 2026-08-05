package task

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestHandlerNotFound(t *testing.T) {
	response := httptest.NewRecorder()
	newTestHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/tasks/missing", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func newTestHandler() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(NewService(&MemoryStore{}), logger)
}
