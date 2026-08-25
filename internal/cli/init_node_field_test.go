package cli

import (
	"strings"
	"testing"
)

// The Node field is prefilled like the PHP field above it: a saved pin is the
// answer to beat, and with none the box shows what the project already resolves
// to rather than sitting empty.
func TestNodeVersionDefault(t *testing.T) {
	cases := []struct{ saved, resolved, want string }{
		{"18", "22", "18"},
		{"", "22", "22"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := nodeVersionDefault(c.saved, c.resolved); got != c.want {
			t.Errorf("nodeVersionDefault(%q, %q) = %q, want %q", c.saved, c.resolved, got, c.want)
		}
	}
}

// Clearing the field writes no pin, which leaves the project following its own
// files. The description has to say that rather than claim the field skips Node.
func TestNodeVersionDescription_NamesWhatClearingDoes(t *testing.T) {
	got := nodeVersionDescription(".nvmrc")
	if !strings.Contains(got, ".nvmrc") {
		t.Errorf("description %q should name what an empty field follows", got)
	}
	if strings.Contains(strings.ToLower(got), "skip") {
		t.Errorf("description %q must not claim an empty field skips Node", got)
	}
}

// Nothing resolved is a case the wizard still has to describe, without naming a
// source it does not have.
func TestNodeVersionDescription_NoSource(t *testing.T) {
	got := nodeVersionDescription("")
	if got == "" || strings.Contains(strings.ToLower(got), "skip") {
		t.Errorf("description %q should describe an empty field without claiming a skip", got)
	}
}
