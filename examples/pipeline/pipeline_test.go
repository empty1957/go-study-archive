package pipeline

import (
	"context"
	"sort"
	"testing"
	"time"
)

func TestMap(t *testing.T) {
	results := Map(context.Background(), 2, []int{1, 2, 3}, func(v int) int { return v * v })
	var got []int
	for value := range results {
		got = append(got, value)
	}
	sort.Ints(got)
	want := []int{1, 4, 9}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Map() = %v, want %v", got, want)
		}
	}
}

func TestMapCancellationClosesOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	results := Map(ctx, 2, []int{1, 2, 3, 4}, func(v int) int { return v })
	cancel()

	timeout := time.After(time.Second)
	for {
		select {
		case _, ok := <-results:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("results channel did not close after cancellation")
		}
	}
}
