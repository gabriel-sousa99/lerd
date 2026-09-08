package config

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// The base image workflow carries its own copy of the version list and its own
// copy of the prerelease exception, and nothing at build time can notice they
// drifted: a version missing from the matrix simply never gets a pre-built base
// and every install falls back to a slow from-source build.
func TestBaseImagesWorkflowCoversSupportedVersions(t *testing.T) {
	b, err := os.ReadFile("../../.github/workflows/base-images.yml")
	if err != nil {
		t.Fatalf("reading base-images.yml: %v", err)
	}
	workflow := string(b)

	quoted := make([]string, len(SupportedPHPVersions))
	for i, v := range SupportedPHPVersions {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	matrix := "php: [" + strings.Join(quoted, ", ") + "]"
	if !strings.Contains(workflow, matrix) {
		t.Errorf("base-images.yml has no build matrix listing every supported version\nwant: %s", matrix)
	}

	single := make([]string, len(PrereleasePHPVersions))
	for i, v := range PrereleasePHPVersions {
		single[i] = "'" + v + "'"
	}
	prerelease := "PRERELEASE = {" + strings.Join(single, ", ") + "}"
	if !strings.Contains(workflow, prerelease) {
		t.Errorf("base-images.yml preprocessing does not resolve the same prereleases to their -rc tag\nwant: %s", prerelease)
	}
}
