package config

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The guard is what stands between a forgetful test and the developer's own lerd
// state, so pin both halves: it fires on a write into the real dirs, and stays out
// of the way of an isolated one.
func TestGuardRealWrite(t *testing.T) {
	if len(realStateDirs) == 0 || realStateDirs[0] == "" {
		t.Skip("no resolvable state dirs in this environment")
	}

	t.Run("panics on a write into the real state dir", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("writing the real sites.yaml from a test must panic, not silently land")
			}
			if msg, _ := r.(string); !strings.Contains(msg, "real lerd state") {
				t.Errorf("panic should name the problem, got %v", r)
			}
		}()
		guardRealWrite(filepath.Join(realStateDirs[0], "sites.yaml"))
	})

	t.Run("allows an isolated write", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("an isolated temp dir must not trip the guard, got %v", r)
			}
		}()
		guardRealWrite(filepath.Join(t.TempDir(), "lerd", "sites.yaml"))
	})
}

// The macOS unit dir follows HOME, not any XDG var, so a test that isolated only
// the XDG vars still wrote real launchd units and left a developer without an
// nginx. The guard has to know that dir the way it knows the Linux ones.
func TestGuardRealWrite_coversTheLaunchAgentsDir(t *testing.T) {
	if runtime.GOOS != "darwin" {
		if LaunchAgentsDir() != "" {
			t.Errorf("launchd units are a macOS path, got %q", LaunchAgentsDir())
		}
		t.Skip("no launchd on this platform")
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("writing a real launchd unit from a test must panic, not silently land")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "real lerd state") {
			t.Errorf("panic should name the problem, got %v", r)
		}
		// The XDG advice is useless here, so the hint has to name the var that
		// actually moves this dir.
		if !strings.Contains(msg, `t.Setenv("HOME"`) {
			t.Errorf("panic should point at HOME for a launchd unit, got %v", r)
		}
	}()
	guardRealWrite(filepath.Join(LaunchAgentsDir(), "lerd-nginx.plist"))
}

// An isolated HOME is what a well-behaved test sets, and it must stay allowed.
func TestGuardRealWrite_allowsAnIsolatedLaunchAgentsDir(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("no launchd on this platform")
	}
	t.Setenv("HOME", t.TempDir())
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a relocated HOME must not trip the guard, got %v", r)
		}
	}()
	guardRealWrite(filepath.Join(LaunchAgentsDir(), "lerd-nginx.plist"))
}

// The dirs are captured at process start, because a test relocates XDG_DATA_HOME
// and a guard that re-resolved them afterwards would compare a temp dir with
// itself and never fire.
func TestGuardRealWrite_ignoresRelocatedXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("relocating XDG mid-test must not disarm the guard")
		}
	}()
	guardRealWrite(filepath.Join(realStateDirs[0], "sites.yaml"))
}
