package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPHPBuildPlanNamesTheVersionAndItsBase(t *testing.T) {
	plan := phpBuildPlan([]string{"8.4"}, false, "explicit rebuild")
	if len(plan) != 1 {
		t.Fatalf("plan has %d items, want 1", len(plan))
	}
	it := plan[0]
	if it.Name != "PHP 8.4 image" || !it.Build || it.Reason != "explicit rebuild" {
		t.Errorf("unexpected item %+v", it)
	}
	if !strings.Contains(it.Ref, "lerd-php84-fpm-base:") {
		t.Errorf("item does not point at the prebuilt base: %q", it.Ref)
	}
}

// A --local build compiles from the upstream image named in the Containerfile,
// so there is no single ref to size up and the plan must not invent one.
func TestPHPBuildPlanLocalBuildHasNoBaseToSize(t *testing.T) {
	plan := phpBuildPlan([]string{"8.4"}, true, "requested by lerd fetch")
	if plan[0].Ref != "" {
		t.Errorf("local build disclosed a base ref: %q", plan[0].Ref)
	}
}

func TestReportPendingDownloadsSaysSoWhenThereIsNothingToDo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var buf bytes.Buffer
	ReportPendingDownloads(&buf)
	if !strings.Contains(buf.String(), "Nothing to download") {
		t.Errorf("unexpected report: %q", buf.String())
	}
}

func TestStartCmdOffersADryRun(t *testing.T) {
	if NewStartCmd().Flags().Lookup("dry-run") == nil {
		t.Error("lerd start has no --dry-run flag")
	}
}
