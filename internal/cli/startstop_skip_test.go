package cli

import (
	"slices"
	"testing"
)

func TestDropSkippedRemovesOnlyTheNamedUnits(t *testing.T) {
	units := []string{"lerd-nginx", "lerd-ui", "lerd-watcher", "lerd-php84-fpm"}

	got := dropSkipped(slices.Clone(units), []string{"lerd-ui"})

	want := []string{"lerd-nginx", "lerd-watcher", "lerd-php84-fpm"}
	if !slices.Equal(got, want) {
		t.Errorf("dropSkipped() = %v, want %v", got, want)
	}
}

func TestDropSkippedWithNothingToSkipIsTheSameList(t *testing.T) {
	units := []string{"lerd-nginx", "lerd-ui"}

	if got := dropSkipped(slices.Clone(units), nil); !slices.Equal(got, units) {
		t.Errorf("dropSkipped() = %v, want %v", got, units)
	}
}
