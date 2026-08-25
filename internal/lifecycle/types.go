package lifecycle

import (
	"errors"
	"io"
	"sync"
)

// Job represents a unit of work that can be run in parallel.
type Job struct {
	Label string
	Run   func(w io.Writer) error
}

// ParallelRunner executes jobs in parallel.
type ParallelRunner func(jobs []Job) error

// SimpleRunner runs jobs concurrently with no spinner UI, for daemons tearing
// lerd down in the background. Every job runs even when an earlier one fails: a
// unit that refuses to stop must not strand the units behind it.
func SimpleRunner(jobs []Job) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, j := range jobs {
		job := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := job.Run(io.Discard); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}
