package lifecycle

import (
	"errors"
	"io"
	"sync/atomic"
	"testing"
)

// TestSimpleRunner_RunsEveryJobDespiteFailures pins that a unit which refuses
// to stop does not strand the units behind it. The daemon shutdown path has one
// pass at the teardown, so bailing on the first error would leave containers up.
func TestSimpleRunner_RunsEveryJobDespiteFailures(t *testing.T) {
	var ran atomic.Int32
	boom := errors.New("boom")
	jobs := []Job{
		{Label: "a", Run: func(io.Writer) error { ran.Add(1); return boom }},
		{Label: "b", Run: func(io.Writer) error { ran.Add(1); return nil }},
		{Label: "c", Run: func(io.Writer) error { ran.Add(1); return boom }},
	}

	err := SimpleRunner(jobs)
	if got := ran.Load(); got != 3 {
		t.Errorf("ran %d jobs, want all 3", got)
	}
	if !errors.Is(err, boom) {
		t.Errorf("SimpleRunner err = %v, want the job failures joined in", err)
	}
}

func TestSimpleRunner_NoJobsNoError(t *testing.T) {
	if err := SimpleRunner(nil); err != nil {
		t.Errorf("SimpleRunner(nil) = %v, want nil", err)
	}
}
