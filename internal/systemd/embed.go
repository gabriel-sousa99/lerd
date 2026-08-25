package systemd

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

//go:embed units
var unitsFS embed.FS

// lerdBinaryPath resolves the absolute path to the running lerd binary so unit
// ExecStart lines point at wherever lerd is actually installed: ~/.local/bin
// for curl/brew, /usr/bin for the deb. A var so tests can override it.
var lerdBinaryPath = config.LerdBinary

// GetUnit returns the content of an embedded systemd unit file with the lerd
// binary path resolved. The templates ship with ExecStart=%h/.local/bin/lerd,
// which only works for a ~/.local/bin install; substituting the real path lets
// the daemon units run from any install location (notably /usr/bin under the
// Debian package).
func GetUnit(name string) (string, error) {
	// name may or may not have .service suffix
	filename := name
	if !strings.HasSuffix(filename, ".service") {
		filename += ".service"
	}
	data, err := unitsFS.ReadFile("units/" + filename)
	if err != nil {
		return "", fmt.Errorf("systemd unit %q not found: %w", name, err)
	}
	return resolveUnitBinaryPath(string(data)), nil
}

// resolveUnitBinaryPath swaps the hardcoded ~/.local/bin templates for the
// running binary's real location. lerd-tray is replaced first because its name
// has lerd as a prefix. A path that isn't absolute would give systemd an
// ExecStart it can't run, so the template default is left in place instead.
func resolveUnitBinaryPath(content string) string {
	bin := lerdBinaryPath()
	if !filepath.IsAbs(bin) {
		return content
	}
	tray := filepath.Join(filepath.Dir(bin), "lerd-tray")
	content = strings.ReplaceAll(content, "%h/.local/bin/lerd-tray", tray)
	content = strings.ReplaceAll(content, "%h/.local/bin/lerd", bin)
	return content
}
