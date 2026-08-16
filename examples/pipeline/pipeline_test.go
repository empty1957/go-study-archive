package pipeline

import (
	"context"
	"sort"
	"testing"
	"time"
)

func TestMap(t *testing.T) {
	results := Map(context.Background(), 2, []int{1, 2, 3}, func(_ context.Context, v int) int { return v * v })
	var got []int
	for value := range results {
		got = append(got, value)
	}
	sort.Ints(got)
	want := []int{1, 4, 9}
	if len(got) != len(want) {
		t.Fatalf("Map() returned %d values, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Map() = %v, want %v", got, want)
		}
	}
}

func TestMapCancellationClosesOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	returned := make(chan struct{})
	results := Map(ctx, 1, []int{1}, func(ctx context.Context, v int) int {
		close(entered)
		<-ctx.Done()
		close(returned)
		return v
	})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("worker callback did not start")
	}
	cancel()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("worker callback did not observe cancellation")
	}

	select {
	case _, ok := <-results:
		if ok {
			t.Fatal("results channel yielded a value after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("results channel did not close after workers joined")
	}
}
