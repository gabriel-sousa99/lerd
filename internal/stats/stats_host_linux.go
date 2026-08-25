//go:build linux

package stats

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// hostCmdTimeout bounds each systemctl call so a wedged systemd never hangs the
// stats read.
const hostCmdTimeout = 3 * time.Second

// cpuPrimeGap is the one short window a process pays on its very first read,
// where no earlier counter exists to measure against. Without it the first
// snapshot after a restart would be a column of zeroes, which reads as broken.
var cpuPrimeGap = 250 * time.Millisecond

// readHostProcesses reports the resource usage of every lerd unit: the containers,
// each of which runs as a quadlet unit, and lerd's own daemons (ui, watcher, tray)
// and host-side workers such as a Vite dev server run via fnm. Everything is read
// from the units' cgroups, so one pass over a handful of files measures the whole
// list the same way. Memory is what the unit holds with page cache excluded, CPU
// is the time it used since the previous read. Linux only; the macOS stub returns
// nothing and Read falls back to podman there.
func readHostProcesses() ([]ContainerStat, error) {
	units := listLerdServices()
	if len(units) == 0 {
		return nil, nil
	}
	props := showProps(units)
	usage, at := sampleCPU(props), time.Now()
	if noPriorSamples() {
		cpuRates(usage, at)
		time.Sleep(cpuPrimeGap)
		usage, at = sampleCPU(props), time.Now()
	}
	rates := cpuRates(usage, at)

	totalRAM := hostTotalRAM()
	var rows []ContainerStat
	for _, u := range units {
		c, ok := props[u]
		if !ok {
			continue
		}
		memPct := 0.0
		if totalRAM > 0 {
			memPct = float64(c.memBytes) / float64(totalRAM) * 100
		}
		rows = append(rows, ContainerStat{
			Name:       strings.TrimSuffix(u, ".service"),
			CPUPercent: rates[u],
			MemBytes:   c.memBytes,
			MemLimit:   totalRAM,
			MemPercent: memPct,
		})
	}
	return rows, nil
}

// cpuSample is one unit's cumulative CPU time and the moment it was read.
type cpuSample struct {
	usec int64
	at   time.Time
}

var (
	cpuMu   sync.Mutex
	cpuPrev = map[string]cpuSample{}
)

// noPriorSamples reports whether this process has ever sampled the CPU counters.
func noPriorSamples() bool {
	cpuMu.Lock()
	defer cpuMu.Unlock()
	return len(cpuPrev) == 0
}

// sampleCPU reads every unit's cumulative CPU time in one pass over the cgroups.
func sampleCPU(props map[string]hostProps) map[string]int64 {
	usage := make(map[string]int64, len(props))
	for u, p := range props {
		usage[u] = cgroupCPUUsec(p.cgroup, p.cpuNsec)
	}
	return usage
}

// cpuRates turns this read's cumulative counters into a percentage of one core
// per unit, measured against the previous read, and keeps them for the next one.
// Spreading the cost over the interval that actually elapsed is the number a
// person reading the list is after: a sample window opened inside the read only
// ever catches lerd measuring itself. A unit seen for the first time has nothing
// to compare against, and a counter that went backwards is a restart; both report
// zero rather than a spike. Units absent from this read drop out of the map.
func cpuRates(usage map[string]int64, at time.Time) map[string]float64 {
	cpuMu.Lock()
	defer cpuMu.Unlock()
	rates := make(map[string]float64, len(usage))
	next := make(map[string]cpuSample, len(usage))
	for u, cur := range usage {
		if prev, ok := cpuPrev[u]; ok {
			if elapsed := at.Sub(prev.at).Seconds(); elapsed > 0 && cur >= prev.usec {
				rates[u] = float64(cur-prev.usec) / 1e6 / elapsed * 100
			}
		}
		next[u] = cpuSample{usec: cur, at: at}
	}
	cpuPrev = next
	return rates
}

// cgroupCPUUsec returns a unit's cumulative CPU time in microseconds, read from
// its cgroup. Falls back to systemd's own counter (nanoseconds) when the cgroup
// file isn't there, so a unit is never silently reported idle.
func cgroupCPUUsec(cg string, fallbackNsec uint64) int64 {
	if cg != "" {
		if v := readCgroupStatKey(cgroupRoot+cg+"/cpu.stat", "usage_usec"); v > 0 {
			return v
		}
	}
	return int64(fallbackNsec / 1000)
}

// listLerdServices returns the running lerd-prefixed user services: the container
// quadlet units (lerd-mysql.service, …) and lerd's own daemons alike, which
// together are every row the dashboard shows.
func listLerdServices() []string {
	ctx, cancel := context.WithTimeout(context.Background(), hostCmdTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "list-units",
		"--type=service", "--state=running", "--no-legend", "--plain", "--no-pager",
		"lerd-*.service").Output()
	if err != nil {
		return nil
	}
	var units []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if strings.HasPrefix(name, "lerd-") && strings.HasSuffix(name, ".service") {
			units = append(units, name)
		}
	}
	return units
}

type hostProps struct {
	cpuNsec  uint64
	memBytes int64
	cgroup   string
}

// showProps batches one `systemctl show` over all units and parses the per-unit
// CPU and memory counters. Units with accounting disabled report "[not set]"
// (left as zero). Blocks are separated by blank lines and labelled by Id.
func showProps(units []string) map[string]hostProps {
	ctx, cancel := context.WithTimeout(context.Background(), hostCmdTimeout)
	defer cancel()
	args := append([]string{"--user", "show", "-p", "Id", "-p", "CPUUsageNSec", "-p", "MemoryCurrent", "-p", "ControlGroup"}, units...)
	out, err := exec.CommandContext(ctx, "systemctl", args...).Output()
	if err != nil {
		return nil
	}
	res := make(map[string]hostProps, len(units))
	var id string
	var p hostProps
	flush := func() {
		if id != "" {
			// systemd's MemoryCurrent is the raw cgroup memory.current, which counts
			// reclaimable page cache; prefer what the unit really holds so a daemon
			// that reads big files isn't reported holding memory it can release on
			// demand.
			if held, ok := cgroupMemoryHeld(p.cgroup); ok {
				p.memBytes = held
			}
			res[id] = p
		}
		id, p = "", hostProps{}
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "Id":
			id = val
		case "CPUUsageNSec":
			if n, err := strconv.ParseUint(val, 10, 64); err == nil {
				p.cpuNsec = n
			}
		case "MemoryCurrent":
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				p.memBytes = n
			}
		case "ControlGroup":
			p.cgroup = val
		}
	}
	flush()
	return res
}

// cgroupRoot is the cgroup v2 mount point. A var so tests can point the memory
// read at a fixture tree instead of the live hierarchy.
var cgroupRoot = "/sys/fs/cgroup"

// cgroupMemoryHeld returns the memory a unit actually holds: memory.current less
// the page cache the kernel can drop under pressure. Shared memory is counted in
// the file total but cannot be dropped, so it stays in. Subtracting only
// inactive_file (the cAdvisor/k8s working set, and what `podman stats` reports)
// leaves the active cache in, and a poller that re-reads the same files every
// tick has all of its cache on the active list, so fifty megabytes of process
// read as two gigabytes. Returns false when the cgroup v2 files aren't present or
// readable, so the caller falls back to MemoryCurrent.
func cgroupMemoryHeld(cg string) (int64, bool) {
	if cg == "" {
		return 0, false
	}
	base := cgroupRoot + cg
	cur, err := readCgroupInt(base + "/memory.current")
	if err != nil {
		return 0, false
	}
	stat := base + "/memory.stat"
	cache := readCgroupStatKey(stat, "file") - readCgroupStatKey(stat, "shmem")
	held := cur - cache
	if held < 0 {
		held = cur
	}
	return held, true
}

// readCgroupInt reads a single-integer cgroup file (e.g. memory.current).
func readCgroupInt(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
}

// readCgroupStatKey returns one key's value from a "key value" cgroup file
// (e.g. memory.stat). Missing key or unreadable file yields 0.
func readCgroupStatKey(path, key string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, " ")
		if ok && k == key {
			if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
				return n
			}
			return 0
		}
	}
	return 0
}

// hostTotalRAM reads MemTotal from /proc/meminfo (bytes), used as the host
// memory denominator for host-process rows so the dashboard's "% of host" stays
// consistent whether or not any container reported a limit.
func hostTotalRAM() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				return kb * 1024
			}
		}
	}
	return 0
}
