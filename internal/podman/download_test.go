package podman

import "testing"

// An operation with no image to fetch reports a zero value, which the dashboard
// reads as "nothing to download" and never prompts about.
func TestDescribeDownloadWithoutAnImage(t *testing.T) {
	if got := DescribeDownload(""); got != (PendingDownload{}) {
		t.Errorf("DescribeDownload(\"\") = %+v, want zero", got)
	}
}

// The PHP build starts from the prebuilt base, so that is the ref the estimate
// has to name; anything else would size up the wrong download.
func TestPHPDownloadNamesThePrebuiltBase(t *testing.T) {
	got := PHPDownload("8.4")
	if got.Image != PHPBaseImageRef("8.4") {
		t.Errorf("PHPDownload image = %q, want %q", got.Image, PHPBaseImageRef("8.4"))
	}
}
