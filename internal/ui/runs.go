package ui

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// runMaxLines caps how much of a run's output is kept for replay. Composer
// prints thousands of lines; a reattaching page wants the tail of that, not all
// of it, and the buffer is what a reload reads back.
const runMaxLines = 2000

// runMaxLineBytes caps one line of output. A command that writes progress with
// a bare carriage return produces a single line megabytes long; it is emitted
// in pieces this size rather than held whole.
const runMaxLineBytes = 1024 * 1024

// runRetention is how long a finished run stays readable, so a page that
// reloads just as the run ends still finds its result rather than a 404.
const runRetention = 30 * time.Minute

// runStatus is where a run got to.
const (
	runRunning = "running"
	runDone    = "done"
	runFailed  = "failed"
)

// run is one lerd command the dashboard started on the host. It outlives the
// request that started it and the page that was watching: scaffolding is
// minutes of composer, and closing the modal or reloading the tab must not take
// the project down with it.
type run struct {
	ID     string
	Kind   string
	Dir    string
	Label  string
	Argv   []string
	Start  time.Time
	seq    uint64 // start order, so listing is newest-first without clock ties
	cancel context.CancelFunc

	mu       sync.Mutex
	lines    []string
	dropped  int // lines evicted from the head of the buffer
	status   string
	failure  string
	finished time.Time
	waiters  []chan struct{}
}

// runSnapshot is a run as the dashboard sees it.
type runSnapshot struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Dir     string `json:"dir"`
	Label   string `json:"label,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Started int64  `json:"started"`
}

func (r *run) snapshot() runSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return runSnapshot{
		ID:      r.ID,
		Kind:    r.Kind,
		Dir:     r.Dir,
		Label:   r.Label,
		Status:  r.status,
		Error:   r.failure,
		Started: r.Start.Unix(),
	}
}

// append records a line and wakes everyone watching.
func (r *run) append(line string) {
	r.mu.Lock()
	r.lines = append(r.lines, line)
	if len(r.lines) > runMaxLines {
		evict := len(r.lines) - runMaxLines
		r.lines = r.lines[evict:]
		r.dropped += evict
	}
	r.wake()
	r.mu.Unlock()
}

// finish records how the run ended and wakes everyone watching. A command that
// merely exited non-zero says nothing on its own, so the reason is read out of
// what it printed, the way the link modal has always surfaced a failure.
func (r *run) finish(err error) {
	r.mu.Lock()
	if err != nil {
		r.status, r.failure = runFailed, failureMessage(strings.Join(r.lines, "\n"))
		if r.failure == "" {
			r.failure = err.Error()
		}
	} else {
		r.status = runDone
	}
	r.finished = time.Now()
	r.wake()
	r.mu.Unlock()
}

// wake signals every waiter. Callers hold the lock.
func (r *run) wake() {
	for _, ch := range r.waiters {
		close(ch)
	}
	r.waiters = nil
}

// read returns the lines from absolute index `from` on, the next index to read,
// and whether the run has finished. A reader that fell behind the buffer's head
// is moved up to it rather than shown lines that are gone.
func (r *run) read(from int) (lines []string, next int, done bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if from < r.dropped {
		from = r.dropped
	}
	if idx := from - r.dropped; idx < len(r.lines) {
		lines = append(lines, r.lines[idx:]...)
	}
	return lines, r.dropped + len(r.lines), r.status != runRunning
}

// wait returns a channel closed on the run's next line or on its end, so a
// watcher blocks rather than polls.
func (r *run) wait() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan struct{})
	if r.status != runRunning {
		close(ch)
		return ch
	}
	r.waiters = append(r.waiters, ch)
	return ch
}

// runRegistry holds the runs this lerd-ui has started.
type runRegistry struct {
	mu   sync.Mutex
	seq  uint64
	runs map[string]*run
}

func newRunRegistry() *runRegistry {
	return &runRegistry{runs: make(map[string]*run)}
}

// runs is the process-wide registry. lerd-ui is the long-running process the
// dashboard talks to, so a run parked here survives every page in front of it.
var runs = newRunRegistry()

func newRunID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}

// Start runs argv in dir and returns immediately. The command is given a
// context of its own rather than the request's, so it keeps going after the
// response that started it is long gone.
func (reg *runRegistry) Start(kind, dir, label string, argv []string) *run {
	ctx, cancel := context.WithCancel(context.Background())
	r := &run{
		ID:     newRunID(),
		Kind:   kind,
		Dir:    dir,
		Label:  label,
		Argv:   argv,
		Start:  time.Now(),
		status: runRunning,
		cancel: cancel,
	}

	reg.mu.Lock()
	reg.seq++
	r.seq = reg.seq
	reg.runs[r.ID] = r
	reg.mu.Unlock()
	reg.sweep()

	go func() {
		defer cancel()
		r.finish(execRun(ctx, r, argv, dir))
		// The page that started this may be long gone, so how it ended is the
		// daemon's to report.
		notifyRunFinished(r)
	}()
	return r
}

// execRun is the seam tests replace: it runs the command and feeds its output
// to the run line by line.
var execRun = func(ctx context.Context, r *run, argv []string, dir string) error {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		r.append(err.Error())
		return err
	}

	var waitErr error
	waited := make(chan struct{})
	go func() {
		waitErr = cmd.Wait()
		pw.Close() //nolint:errcheck
		close(waited)
	}()

	// Nothing else reads this pipe, so the drain has to reach EOF whatever the
	// command writes: a reader that gave up on a line longer than its buffer
	// would leave the command blocked writing to it and cmd.Wait below never
	// returning. An over-long line is emitted in pieces instead of abandoned.
	reader := bufio.NewReaderSize(pr, 64*1024)
	var line []byte
	for {
		chunk, isPrefix, err := reader.ReadLine()
		if err != nil {
			break
		}
		line = append(line, chunk...)
		if !isPrefix || len(line) >= runMaxLineBytes {
			r.append(string(line))
			line = line[:0]
		}
	}
	if len(line) > 0 {
		r.append(string(line))
	}
	<-waited
	return waitErr
}

// Get returns a run by id.
func (reg *runRegistry) Get(id string) (*run, bool) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	r, ok := reg.runs[id]
	return r, ok
}

// ForDir lists the runs started for a directory, newest first, so a reloaded
// wizard finds the work it left behind rather than starting it again.
func (reg *runRegistry) ForDir(dir string) []runSnapshot {
	// Also swept here, not only when a run starts: a machine that scaffolded one
	// project and was left alone would otherwise hold that run, its buffered
	// output and its place in every listing for as long as lerd-ui is up.
	reg.sweep()

	reg.mu.Lock()
	all := make([]*run, 0, len(reg.runs))
	for _, r := range reg.runs {
		if dir == "" || r.Dir == dir {
			all = append(all, r)
		}
	}
	reg.mu.Unlock()

	sort.Slice(all, func(i, j int) bool { return all[i].seq > all[j].seq })
	out := make([]runSnapshot, 0, len(all))
	for _, r := range all {
		out = append(out, r.snapshot())
	}
	return out
}

// sweep drops finished runs once they are old enough that nothing is coming
// back for them.
func (reg *runRegistry) sweep() {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	for id, r := range reg.runs {
		r.mu.Lock()
		expired := r.status != runRunning && !r.finished.IsZero() && time.Since(r.finished) > runRetention
		r.mu.Unlock()
		if expired {
			delete(reg.runs, id)
		}
	}
}
