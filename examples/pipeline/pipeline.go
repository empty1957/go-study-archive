// Package pipeline demonstrates bounded concurrency and cancellation.
package pipeline

import (
	"context"
	"sync"
)

// Map applies fn with at most workers concurrent calls. Output order is not
// guaranteed. fn must observe ctx when it performs blocking work.
//
// The caller owns cancellation: if it stops receiving before results is
// closed, it must cancel ctx. Map owns and closes its internal channels, and
// the goroutine that waits for all workers is the only closer of results.
func Map[I, O any](ctx context.Context, workers int, input []I, fn func(context.Context, I) O) <-chan O {
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
				result := fn(ctx, value)
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
