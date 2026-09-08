package serviceops

import "testing"

func TestRetagImage(t *testing.T) {
	cases := []struct{ image, tag, want string }{
		{"docker.io/library/mysql:8.0", "8.4", "docker.io/library/mysql:8.4"},
		{"mysql", "8.4", "mysql:8.4"},
		// A registry port is not a tag, so the tag is appended rather than replacing it.
		{"registry.local:5000/team/mysql", "8.4", "registry.local:5000/team/mysql:8.4"},
		{"docker.io/library/mysql:8.0", "", "docker.io/library/mysql:8.0"},
		{"", "8.4", ""},
	}
	for _, tc := range cases {
		if got := RetagImage(tc.image, tc.tag); got != tc.want {
			t.Errorf("RetagImage(%q, %q) = %q, want %q", tc.image, tc.tag, got, tc.want)
		}
	}
}

func TestActionDownloadRejectsAnUnknownAction(t *testing.T) {
	if _, err := ActionDownload("mysql", "frobnicate", ""); err == nil {
		t.Error("an unknown action was accepted")
	}
}
