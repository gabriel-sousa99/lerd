package cli

import (
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// The runtime a re-routed command needs is installed by a command the
// declaration names, because two packages that both escape the container do not
// install the same way: nativephp/mobile installs through native:install-mobile
// precisely because the desktop package already owns native:install (#1651).
func TestHostCommandMissingMsg_namesTheDeclaredInstallCommand(t *testing.T) {
	hc := config.HostCommand{
		Args:           "artisan native:*",
		Binary:         "vendor/nativephp/php-bin/bin/host/php",
		InstallCommand: "native:install-mobile",
	}

	msg := hostCommandMissingMsg("artisan", hc)
	if !strings.Contains(msg, "vendor/nativephp/php-bin/bin/host/php") {
		t.Errorf("message = %q, want it to name the missing binary", msg)
	}
	if !strings.Contains(msg, "lerd run native:install-mobile") {
		t.Errorf("message = %q, want it to name the declared install command", msg)
	}
}

// A declaration that names no install command is not guessed at, since guessing
// is how a mobile project was told to run the desktop installer.
func TestHostCommandMissingMsg_genericWhenNoneDeclared(t *testing.T) {
	hc := config.HostCommand{Args: "artisan native:*", Binary: "vendor/bin/runtime"}

	msg := hostCommandMissingMsg("artisan", hc)
	if !strings.Contains(msg, "vendor/bin/runtime") {
		t.Errorf("message = %q, want it to name the missing binary", msg)
	}
	if strings.Contains(msg, "native:install") {
		t.Errorf("message = %q, must not name a command no declaration carries", msg)
	}
	if strings.Contains(msg, "lerd run ") {
		t.Errorf("message = %q, must not point at a command it cannot name", msg)
	}
}
