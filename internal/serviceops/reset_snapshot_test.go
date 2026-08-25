package serviceops

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// stubResetSnapshot swaps the pre-reset snapshot seam for a recorder, so the
// ordering and the label each operation asks for can be asserted without a
// running engine.
type resetSnapshotRecorder struct {
	calls  []string
	labels []string
	err    error
}

func stubResetSnapshot(t *testing.T, order *[]string) *resetSnapshotRecorder {
	t.Helper()
	rec := &resetSnapshotRecorder{}
	prev := removeSnapshotFn
	removeSnapshotFn = func(name, label string, _ func(PhaseEvent)) error {
		rec.calls = append(rec.calls, name)
		rec.labels = append(rec.labels, label)
		if order != nil {
			*order = append(*order, "snapshot")
		}
		return rec.err
	}
	t.Cleanup(func() { removeSnapshotFn = prev })
	return rec
}

func TestRemoveServiceSnapshotsTheDatabasesBeforeWipingData(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	podRec := stubPodmanRemove(t)
	var order []string
	rec := stubResetSnapshot(t, &order)
	mkTestDataDir(t, "mysql", "user-data")

	prevStop := removeStopUnit
	removeStopUnit = func(name string) error {
		order = append(order, "stop")
		podRec.stopped = append(podRec.stopped, name)
		return nil
	}
	t.Cleanup(func() { removeStopUnit = prevStop })

	if err := RemoveService("mysql", RemoveOptions{RemoveData: true}, func(PhaseEvent) {}); err != nil {
		t.Fatalf("RemoveService: %v", err)
	}

	if len(rec.calls) != 1 || rec.calls[0] != "mysql" {
		t.Fatalf("expected one snapshot of mysql, got %v", rec.calls)
	}
	if rec.labels[0] != "pre-remove" {
		t.Errorf("snapshot label = %q, want pre-remove", rec.labels[0])
	}
	if len(order) < 2 || order[0] != "snapshot" {
		t.Errorf("the snapshot must be taken before the engine is stopped, got %v", order)
	}
}

func TestRemoveServiceKeepingDataTakesNoSnapshot(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	stubPodmanRemove(t)
	rec := stubResetSnapshot(t, nil)
	mkTestDataDir(t, "mysql", "user-data")

	if err := RemoveService("mysql", RemoveOptions{RemoveData: false}, func(PhaseEvent) {}); err != nil {
		t.Fatalf("RemoveService: %v", err)
	}
	if len(rec.calls) != 0 {
		t.Errorf("nothing is wiped without RemoveData, so no snapshot is due, got %v", rec.calls)
	}
}

func TestRemoveServiceSkipSnapshotOptOut(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	stubPodmanRemove(t)
	rec := stubResetSnapshot(t, nil)
	dir := mkTestDataDir(t, "mysql", "user-data")

	if err := RemoveService("mysql", RemoveOptions{RemoveData: true, SkipSnapshot: true}, func(PhaseEvent) {}); err != nil {
		t.Fatalf("RemoveService: %v", err)
	}
	if len(rec.calls) != 0 {
		t.Errorf("SkipSnapshot must suppress the snapshot, got %v", rec.calls)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the data dir should still have been renamed aside, stat err = %v", err)
	}
}

func TestRemoveServiceStopsWhenTheSnapshotFails(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	podRec := stubPodmanRemove(t)
	rec := stubResetSnapshot(t, nil)
	rec.err = errors.New("engine never became ready")
	dir := mkTestDataDir(t, "mysql", "user-data")

	err := RemoveService("mysql", RemoveOptions{RemoveData: true}, func(PhaseEvent) {})
	if err == nil {
		t.Fatal("expected the removal to fail when its safety snapshot could not be taken")
	}
	if !strings.Contains(err.Error(), "engine never became ready") {
		t.Errorf("error should carry the snapshot failure, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "marker.txt")); statErr != nil {
		t.Errorf("data must be untouched when the snapshot fails: %v", statErr)
	}
	if len(podRec.stopped) != 0 {
		t.Errorf("the service should not have been stopped, got %v", podRec.stopped)
	}
}

func TestReinstallServiceLabelsTheSnapshotForItsReset(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	stubPodmanRemove(t)
	stubReinstallSeams(t)
	rec := stubResetSnapshot(t, nil)
	saveCustomServiceForReinstall(t, "mariadb", "11.8")

	if err := ReinstallService("mariadb", ReinstallOptions{ResetData: true}, func(PhaseEvent) {}); err != nil {
		t.Fatalf("ReinstallService: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("a data reset should snapshot first, got %v", rec.calls)
	}
	if rec.labels[0] != "pre-reset-data" {
		t.Errorf("snapshot label = %q, want pre-reset-data", rec.labels[0])
	}
}

func TestReinstallServiceSkipSnapshotOptOut(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	stubPodmanRemove(t)
	stubReinstallSeams(t)
	rec := stubResetSnapshot(t, nil)
	saveCustomServiceForReinstall(t, "mariadb", "11.8")

	if err := ReinstallService("mariadb", ReinstallOptions{ResetData: true, SkipSnapshot: true}, func(PhaseEvent) {}); err != nil {
		t.Fatalf("ReinstallService: %v", err)
	}
	if len(rec.calls) != 0 {
		t.Errorf("SkipSnapshot must reach the remove step, got %v", rec.calls)
	}
}

func TestSnapshotBeforeDataResetSkipsAServiceWithoutSnapshots(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	mkTestDataDir(t, "redis", "cache")

	if err := snapshotBeforeDataReset("redis", "pre-remove", func(PhaseEvent) {}); err != nil {
		t.Fatalf("a service that declares no dump has nothing to snapshot: %v", err)
	}
}

func TestSnapshotBeforeDataResetSkipsWhenThereIsNoData(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	if _, err := os.Stat(config.DataSubDir("mysql")); !os.IsNotExist(err) {
		t.Fatalf("fixture expects no data dir, stat err = %v", err)
	}
	if err := snapshotBeforeDataReset("mysql", "pre-remove", func(PhaseEvent) {}); err != nil {
		t.Fatalf("there is nothing to lose without a data dir: %v", err)
	}
}
