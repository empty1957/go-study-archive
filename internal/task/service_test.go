package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		want    string
		wantErr error
	}{
		{name: "trims title", title: "  learn context  ", want: "learn context"},
		{name: "empty", title: "  ", wantErr: ErrInvalidTitle},
		{name: "too long", title: strings.Repeat("界", 201), wantErr: ErrInvalidTitle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&MemoryStore{})
			service.now = func() time.Time { return time.Date(2026, 8, 6, 1, 2, 3, 0, time.UTC) }

			got, err := service.Create(context.Background(), tt.title)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.Title != tt.want {
				t.Errorf("Title = %q, want %q", got.Title, tt.want)
			}
			if got.ID == "" {
				t.Error("ID must not be empty")
			}
		})
	}
}

func TestMemoryStoreHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := &MemoryStore{}
	err := store.Create(ctx, Task{ID: "1"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
}

func TestServicePreservesNotFound(t *testing.T) {
	service := NewService(&MemoryStore{})
	_, err := service.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want wrapped ErrNotFound", err)
	}
}

func TestMemoryStoreConcurrentUse(t *testing.T) {
	store := &MemoryStore{}
	const count = 50
	done := make(chan struct{}, count)

	for i := 0; i < count; i++ {
		go func(id string) {
			defer func() { done <- struct{}{} }()
			_ = store.Create(context.Background(), Task{ID: id})
		}(string(rune('A' + i)))
	}
	for i := 0; i < count; i++ {
		<-done
	}

	items, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != count {
		t.Fatalf("List() returned %d tasks, want %d", len(items), count)
	}
}
