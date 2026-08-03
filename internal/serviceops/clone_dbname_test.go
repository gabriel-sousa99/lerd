package serviceops

import (
	"strings"
	"testing"
)

// TestValidateDatabaseName_RejectsInjection proves that database names capable
// of breaking out of a shell command (the CloneDatabase injection vector) are
// rejected before any command is composed. Every declared entity command is
// expanded through expandEntityCommand, which gates on this validator.
func TestValidateDatabaseName_RejectsInjection(t *testing.T) {
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
		if err := ValidateDatabaseName(name); err == nil {
			t.Errorf("ValidateDatabaseName(%q) = nil; want error (injection / invalid name must be rejected)", name)
		}
	}
}

// TestValidateDatabaseName_AcceptsLegitimate proves that legitimate database
// names are accepted unchanged.
func TestValidateDatabaseName_AcceptsLegitimate(t *testing.T) {
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
		if err := ValidateDatabaseName(name); err != nil {
			t.Errorf("ValidateDatabaseName(%q) = %v; want nil (legitimate name must be accepted)", name, err)
		}
	}
}

// TestExpandEntityCommand_RejectsMaliciousNames proves a malicious name never
// reaches a composed shell command, which is the fail-fast CloneDatabase relies
// on for both its src and its dst.
func TestExpandEntityCommand_RejectsMaliciousNames(t *testing.T) {
	for _, name := range []string{"x|touch /tmp/pwn", "a$(b)"} {
		if _, err := expandEntityCommand("mysqldump {{name}}", name); err == nil {
			t.Errorf("expandEntityCommand with malicious name %q returned nil error; want rejection", name)
		}
	}
}
