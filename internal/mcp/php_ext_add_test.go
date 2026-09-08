package mcp

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The MCP path takes the same guard as the CLI: declaring an extension the
// image already ships rebuilds it generically on top of the base image and
// loses the configure flags that build gave it (#1576).
func TestExecPHPExtAdd_RejectsABundledExtension(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	res, rpcErr := execPHPExtAdd(map[string]any{"extension": "ftp", "version": "8.5"})
	if rpcErr != nil {
		t.Fatalf("execPHPExtAdd: %v", rpcErr)
	}
	text, isErr := textOf(t, res)
	if !isErr || !strings.Contains(text, "already ships") {
		t.Fatalf("adding a bundled extension must fail with an 'already ships' message, got isError=%v: %s", isErr, text)
	}

	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	if slices.Contains(cfg.GetExtensions(), "ftp") {
		t.Errorf("a rejected add must not declare the extension: %v", cfg.GetExtensions())
	}
}
