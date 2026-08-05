package task

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var ErrInvalidTitle = errors.New("title must contain between 1 and 200 characters")

type Service struct {
	store Store
	now   func() time.Time
	next  atomic.Uint64
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

// Create validates a domain rule before crossing the storage boundary.
func (s *Service) Create(ctx context.Context, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 200 {
		return Task{}, ErrInvalidTitle
	}

	item := Task{
		ID:        strconv.FormatUint(s.next.Add(1), 10),
		Title:     title,
		CreatedAt: s.now().UTC(),
	}
	if err := s.store.Create(ctx, item); err != nil {
		return Task{}, fmt.Errorf("store task %q: %w", item.ID, err)
	}
	return item, nil
}

func (s *Service) Get(ctx context.Context, id string) (Task, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return Task{}, fmt.Errorf("get task %q: %w", id, err)
	}
	return item, nil
}

func (s *Service) List(ctx context.Context) ([]Task, error) {
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return items, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task %q: %w", id, err)
	}
	return nil
}
