package task

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var ErrNotFound = errors.New("task not found")

// Store is defined by its consumer and exposes only the operations the service
// needs. Context is accepted now so an I/O-backed implementation can honor
// cancellation without changing the interface.
type Store interface {
	Create(context.Context, Task) error
	Get(context.Context, string) (Task, error)
	List(context.Context) ([]Task, error)
	Delete(context.Context, string) error
}

// MemoryStore is a concurrency-safe Store for examples and tests.
// Its zero value is ready for use.
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Task
}

func (s *MemoryStore) Create(ctx context.Context, item Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		s.items = make(map[string]Task)
	}
	s.items[item.ID] = item
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return item, nil
}

func (s *MemoryStore) List(ctx context.Context) ([]Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	items := make([]Task, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	s.mu.RUnlock()

	// Map iteration order is deliberately unspecified. Stable API output makes
	// clients and tests deterministic.
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}
