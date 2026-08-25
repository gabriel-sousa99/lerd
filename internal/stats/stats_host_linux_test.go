//go:build linux

package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// resetCPUSamples clears the counters carried between reads so each test starts
// from a process that has never sampled anything.
func resetCPUSamples(t *testing.T) {
	t.Helper()
	cpuMu.Lock()
	cpuPrev = map[string]cpuSample{}
	cpuMu.Unlock()
	t.Cleanup(func() {
		cpuMu.Lock()
		cpuPrev = map[string]cpuSample{}
		cpuMu.Unlock()
	})
}

func TestReadCgroupStatKey(t *testing.T) {
	dir := t.TempDir()
	stat := filepath.Join(dir, "memory.stat")
	body := "anon 24117248\nfile 664797184\ninactive_file 600000000\nkernel 38797312\n"
	if err := os.WriteFile(stat, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readCgroupStatKey(stat, "inactive_file"); got != 600000000 {
		t.Errorf("inactive_file = %d, want 600000000", got)
	}
	if got := readCgroupStatKey(stat, "anon"); got != 24117248 {
		t.Errorf("anon = %d, want 24117248", got)
	}
	if got := readCgroupStatKey(stat, "missing"); got != 0 {
		t.Errorf("missing key = %d, want 0", got)
	}
	if got := readCgroupStatKey(filepath.Join(dir, "nope"), "anon"); got != 0 {
		t.Errorf("unreadable file = %d, want 0", got)
	}
}

// fakeCgroup writes a memory.current/memory.stat pair into a fixture tree and
// points cgroupRoot at it, returning the unit's cgroup path.
func fakeCgroup(t *testing.T, current, stat string) string {
	t.Helper()
	root := t.TempDir()
	cg := "/user.slice/user@1000.service/app.slice/lerd-ui.service"
	dir := filepath.Join(root, cg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "memory.current"), []byte(current), 0o644); err != nil {
		t.Fatal(err)
	}
	if stat != "" {
		if err := os.WriteFile(filepath.Join(dir, "memory.stat"), []byte(stat), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	prev := cgroupRoot
	cgroupRoot = root
	t.Cleanup(func() { cgroupRoot = prev })
	return cg
}

// The numbers are a real lerd-ui after three hours: 2 GB of memory.current, all
// of it page cache the poller read twice so none of it is on the inactive list.
// Subtracting only inactive_file leaves the whole 2 GB in the row.
func TestCgroupMemoryHeld_ExcludesActiveCacheToo(t *testing.T) {
	cg := fakeCgroup(t, "2048086016\n",
		"anon 24117248\nfile 1942446080\nkernel 73400320\nshmem 0\ninactive_file 0\nactive_file 1942446080\n")

	held, ok := cgroupMemoryHeld(cg)
	if !ok {
		t.Fatal("cgroupMemoryHeld reported no data for a populated cgroup")
	}
	if want := int64(105639936); held != want {
		t.Errorf("held = %d, want %d (current - file, not current - inactive_file)", held, want)
	}
}

// Shared memory is counted inside the file total but the kernel cannot drop it,
// so it has to stay in the number a service is reported holding.
func TestCgroupMemoryHeld_KeepsSharedMemory(t *testing.T) {
	cg := fakeCgroup(t, "1000000000\n",
		"anon 600000000\nfile 380000000\nshmem 80000000\nkernel 20000000\ninactive_file 300000000\n")

	held, ok := cgroupMemoryHeld(cg)
	if !ok {
		t.Fatal("cgroupMemoryHeld reported no data for a populated cgroup")
	}
	if want := int64(700000000); held != want {
		t.Errorf("held = %d, want %d (shmem stays counted)", held, want)
	}
}

// memory.current and memory.stat are read separately and can disagree under a
// racing reclaim; never report a negative row.
func TestCgroupMemoryHeld_FallsBackWhenStatOutrunsCurrent(t *testing.T) {
	cg := fakeCgroup(t, "100000\n", "file 900000\nshmem 0\n")

	held, ok := cgroupMemoryHeld(cg)
	if !ok || held != 100000 {
		t.Errorf("held = %d ok = %v, want 100000 true (fall back to memory.current)", held, ok)
	}
}

// CPU comes from the cumulative counter across consecutive reads, so the rate
// covers the interval that actually elapsed, including the idle stretch between
// refreshes, rather than a window opened inside the read itself.
func TestCPURates_MeasureTheIntervalBetweenReads(t *testing.T) {
	resetCPUSamples(t)
	at := time.Unix(1_000_000, 0)

	// Nothing to compare a first sighting against.
	if got := cpuRates(map[string]int64{"lerd-ui.service": 5_000_000}, at); got["lerd-ui.service"] != 0 {
		t.Errorf("first read = %v, want 0", got["lerd-ui.service"])
	}

	// Half a second of CPU over ten seconds is 5% of one core.
	got := cpuRates(map[string]int64{"lerd-ui.service": 5_500_000}, at.Add(10*time.Second))
	if p := got["lerd-ui.service"]; p < 4.99 || p > 5.01 {
		t.Errorf("rate = %v, want ~5", p)
	}
}

// A restarted unit starts its counter again, which must not read as a spike, and
// a unit that has gone away must not sit in the sample map forever.
func TestCPURates_RestartAndDeparture(t *testing.T) {
	resetCPUSamples(t)
	at := time.Unix(2_000_000, 0)
	cpuRates(map[string]int64{"lerd-queue.service": 9_000_000, "lerd-gone.service": 1}, at)

	got := cpuRates(map[string]int64{"lerd-queue.service": 12_000}, at.Add(10*time.Second))
	if got["lerd-queue.service"] != 0 {
		t.Errorf("restarted unit = %v, want 0", got["lerd-queue.service"])
	}
	if _, ok := cpuPrev["lerd-gone.service"]; ok {
		t.Error("a unit absent from the read should not stay in the sample map")
	}
}

func TestCgroupCPUUsec_PrefersTheCgroupOverSystemd(t *testing.T) {
	cg := fakeCgroup(t, "1000\n", "anon 1000\n")
	dir := filepath.Join(cgroupRoot, cg)
	if err := os.WriteFile(filepath.Join(dir, "cpu.stat"), []byte("usage_usec 121771336\nuser_usec 60824838\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := cgroupCPUUsec(cg, 7_000_000_000); got != 121771336 {
		t.Errorf("usage = %d, want the cgroup's 121771336", got)
	}
	// No cpu.stat (an unreadable or v1 cgroup): systemd's own counter, in
	// nanoseconds, is the fallback.
	if got := cgroupCPUUsec("/nope", 7_000_000_000); got != 7_000_000 {
		t.Errorf("fallback = %d, want 7000000", got)
	}
}

func TestCgroupMemoryHeld_NoData(t *testing.T) {
	if _, ok := cgroupMemoryHeld(""); ok {
		t.Error("empty cgroup path should report ok=false so the caller keeps MemoryCurrent")
	}
	root := t.TempDir()
	prev := cgroupRoot
	cgroupRoot = root
	t.Cleanup(func() { cgroupRoot = prev })
	if _, ok := cgroupMemoryHeld("/user.slice/gone.service"); ok {
		t.Error("missing memory.current should report ok=false")
	}
}
