package cli

import (
	"reflect"
	"testing"
)

// The install used to fetch the current store presets at the very end of its
// run, after the services had already started from the quadlets they had. A
// preset that gained a file mount (phpmyadmin's Apache alias, which the
// dashboard proxy needs) therefore landed on disk with nothing left in the run
// to re-render the unit, so lerd-ui — which reads the preset, not the
// install-time copy — served the dashboard at a path the container knew nothing
// about until the user's next `lerd start`. The refresh has to come first and
// the reconcile has to follow it in the same run.
func TestRefreshPresetsThenReconcile_refreshesBeforeReconciling(t *testing.T) {
	var order []string

	origRefresh, origReconcile := refreshPresetsFn, reconcileServicesFn
	refreshPresetsFn = func() { order = append(order, "refresh") }
	reconcileServicesFn = func() { order = append(order, "reconcile") }
	t.Cleanup(func() { refreshPresetsFn, reconcileServicesFn = origRefresh, origReconcile })

	refreshPresetsThenReconcile()

	if want := []string{"refresh", "reconcile"}; !reflect.DeepEqual(order, want) {
		t.Errorf("ran %v, want %v", order, want)
	}
}
