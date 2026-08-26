package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"
)

const (
	maxRequestBody     = 16 << 10 // 16 KiB
	DefaultMaxInFlight = 64
	retryAfterSeconds  = "1"
)

type Handler struct {
	service  *Service
	logger   *slog.Logger
	ready    func() bool
	inFlight chan struct{}
}

func NewHandler(service *Service, logger *slog.Logger) http.Handler {
	return NewHandlerWithReadiness(service, logger, func() bool { return true })
}

// NewHandlerWithReadiness lets the process lifecycle remove this handler from
// service before graceful shutdown begins. ready must be safe for concurrent
// use because each request runs in its own goroutine.
func NewHandlerWithReadiness(service *Service, logger *slog.Logger, ready func() bool) http.Handler {
	return NewHandlerWithReadinessAndLimit(service, logger, ready, DefaultMaxInFlight)
}

// NewHandlerWithReadinessAndLimit bounds the number of API requests admitted
// to the service at once. Health endpoints bypass the limit so overload remains
// observable. maxInFlight must be positive.
func NewHandlerWithReadinessAndLimit(service *Service, logger *slog.Logger, ready func() bool, maxInFlight int) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if ready == nil {
		ready = func() bool { return true }
	}
	if maxInFlight <= 0 {
		panic("task: maxInFlight must be positive")
	}
	h := &Handler{
		service:  service,
		logger:   logger,
		ready:    ready,
		inFlight: make(chan struct{}, maxInFlight),
	}
	api := http.NewServeMux()
	api.HandleFunc("GET /v1/tasks", h.list)
	api.HandleFunc("POST /v1/tasks", h.create)
	api.HandleFunc("GET /v1/tasks/{id}", h.get)
	api.HandleFunc("DELETE /v1/tasks/{id}", h.delete)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /readyz", h.readiness)
	mux.Handle("/v1/", h.admit(api))
	return h.accessLog(mux)
}

type taskResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func responseFromTask(item Task) taskResponse {
	return taskResponse(item)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) readiness(w http.ResponseWriter, _ *http.Request) {
	if !h.ready() {
		h.writeError(w, http.StatusServiceUnavailable, "not_ready", "service is not ready")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		h.writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return
	}
	if r.ContentLength > maxRequestBody {
		h.writeRequestTooLarge(w)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	var input struct {
		Title string `json:"title"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			h.writeRequestTooLarge(w)
			return
		}
		h.writeError(w, http.StatusBadRequest, "invalid_json", "request body must be one valid JSON object")
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			h.writeRequestTooLarge(w)
			return
		}
		h.writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain one JSON object")
		return
	}

	item, err := h.service.Create(r.Context(), input.Title)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			h.writeError(w, http.StatusBadRequest, "invalid_title", ErrInvalidTitle.Error())
			return
		}
		h.internalError(w, r, err)
		return
	}
	w.Header().Set("Location", "/v1/tasks/"+item.ID)
	h.writeJSON(w, http.StatusCreated, responseFromTask(item))
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values")
	}
	return err
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	out := make([]taskResponse, len(items))
	for i, item := range items {
		out[i] = responseFromTask(item)
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleStoreError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, responseFromTask(item))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("id")); err != nil {
		h.handleStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleStoreError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		h.writeError(w, http.StatusNotFound, "not_found", "task was not found")
		return
	}
	h.internalError(w, r, err)
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	h.writeError(w, http.StatusInternalServerError, "internal", "internal server error")
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, errorResponse{Code: code, Message: message})
}

func (h *Handler) writeRequestTooLarge(w http.ResponseWriter) {
	h.writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body exceeds 16 KiB")
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.logger.Error("encode response", "error", err)
	}
}

func (h *Handler) admit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case h.inFlight <- struct{}{}:
			defer func() { <-h.inFlight }()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Retry-After", retryAfterSeconds)
			h.writeError(w, http.StatusServiceUnavailable, "overloaded", "service is temporarily overloaded")
		}
	})
}

func (h *Handler) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		h.logger.InfoContext(r.Context(), "request complete",
			"method", r.Method,
			"path", routePattern(r),
			"duration", time.Since(started),
		)
	})
}

func routePattern(r *http.Request) string {
	switch {
	case r.URL.Path == "/healthz":
		return "/healthz"
	case r.URL.Path == "/readyz":
		return "/readyz"
	case r.URL.Path == "/v1/tasks":
		return "/v1/tasks"
	case strings.HasPrefix(r.URL.Path, "/v1/tasks/"):
		// Log a bounded route label rather than an unbounded task ID.
		return "/v1/tasks/{id}"
	}
	return fmt.Sprintf("unmatched:%s", r.Method)
}
