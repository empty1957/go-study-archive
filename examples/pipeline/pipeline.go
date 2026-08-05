// Package pipeline demonstrates bounded concurrency and cancellation.
package pipeline

import (
	"context"
	"sync"
)

// Map applies fn with at most workers concurrent calls. Output order is not
// guaranteed. Cancellation stops producers and workers without blocked sends.
func Map[I, O any](ctx context.Context, workers int, input []I, fn func(I) O) <-chan O {
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan I)
	results := make(chan O)

	go func() {
		defer close(jobs)
		for _, value := range input {
			select {
			case jobs <- value:
			case <-ctx.Done():
				return
			}
		}
	}()

	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			for value := range jobs {
				result := fn(value)
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		group.Wait()
		close(results)
	}()
	return results
}
