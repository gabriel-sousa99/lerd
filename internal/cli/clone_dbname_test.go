package cli

import (
	"strings"
	"testing"
)

// TestValidDBName_RejectsInjection proves that database names capable of
// breaking out of a shell command (the CloneDatabase injection vector) are
// rejected by validDBName.
func TestValidDBName_RejectsInjection(t *testing.T) {
	malicious := []string{
		"x|touch /tmp/pwn",
		"a`b",
		"a;b",
		"a b",
		"a$(b)",
		"a&b",
		"a>b",
		"\"db\"",
		"a'b",
		"a\nb",
		"",
		strings.Repeat("a", 65), // exceeds 64
	}
	for _, name := range malicious {
		if validDBName(name) {
			t.Errorf("validDBName(%q) = true; want false (injection / invalid name must be rejected)", name)
		}
	}
}

// TestValidDBName_AcceptsLegitimate proves that legitimate database names are
// accepted unchanged.
func TestValidDBName_AcceptsLegitimate(t *testing.T) {
	legit := []string{
		"app",
		"loja_main",
		"db123",
		"meu_app",
		"A",
		"WEB_DEV",
		strings.Repeat("a", 64), // exactly 64
	}
	for _, name := range legit {
		if !validDBName(name) {
			t.Errorf("validDBName(%q) = false; want true (legitimate name must be accepted)", name)
		}
	}
}

// TestCloneDatabase_RejectsMaliciousNames proves CloneDatabase fails fast on a
// malicious src/dst before constructing any shell command.
func TestCloneDatabase_RejectsMaliciousNames(t *testing.T) {
	if err := CloneDatabase("mysql", "x|touch /tmp/pwn", "dst"); err == nil {
		t.Error("CloneDatabase with malicious src returned nil error; want rejection")
	}
	if err := CloneDatabase("mysql", "src", "a$(b)"); err == nil {
		t.Error("CloneDatabase with malicious dst returned nil error; want rejection")
	}
}
