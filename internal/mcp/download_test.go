package mcp

import (
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/podman"
)

func TestDownloadGate(t *testing.T) {
	pull := podman.PendingDownload{Image: "docker.io/library/redis:7", Bytes: 35651584}

	t.Run("discloses the image and its size before anything is fetched", func(t *testing.T) {
		res := downloadGate(map[string]any{}, pull, "Installing the redis preset")
		if res == nil {
			t.Fatal("want a disclosure for an image that is not on this machine")
		}
		text := mcpText(t, res)
		if !strings.Contains(text, "docker.io/library/redis:7") || !strings.Contains(text, "34.0 MiB") {
			t.Errorf("disclosure = %q, want the image and its size", text)
		}
		if !mcpIsError(res) {
			t.Error("the disclosure must not read as a completed operation")
		}
	})

	t.Run("lets the second call through", func(t *testing.T) {
		if res := downloadGate(map[string]any{"confirm": true}, pull, "Installing the redis preset"); res != nil {
			t.Errorf("confirm: true should proceed, got %q", mcpText(t, res))
		}
	})

	// Nothing is spent when the image is already here, so there is nothing to
	// interrupt the assistant for.
	t.Run("says nothing about an image already in the local store", func(t *testing.T) {
		local := podman.PendingDownload{Image: "docker.io/library/redis:7", Local: true}
		if res := downloadGate(map[string]any{}, local, "Installing the redis preset"); res != nil {
			t.Errorf("a local image needs no disclosure, got %q", mcpText(t, res))
		}
	})

	t.Run("says nothing when the operation fetches nothing", func(t *testing.T) {
		if res := downloadGate(map[string]any{}, podman.PendingDownload{}, "Updating redis"); res != nil {
			t.Errorf("an operation with nothing to fetch needs no disclosure, got %q", mcpText(t, res))
		}
	})

	// A registry that does not answer must not turn into a number the user
	// would take for the real cost, and must not block the call either.
	t.Run("still discloses a pull it could not size", func(t *testing.T) {
		res := downloadGate(map[string]any{}, podman.PendingDownload{Image: "ghcr.io/acme/thing:1"}, "Updating thing")
		if res == nil {
			t.Fatal("want a disclosure even without a size")
		}
		if !strings.Contains(mcpText(t, res), "size unknown") {
			t.Errorf("disclosure = %q, want it to say the size is unknown", mcpText(t, res))
		}
	})
}

// Every action that can start a pull has to accept the confirm the disclosure
// asks for, otherwise a strict client drops it and the call never gets past
// the gate.
func TestPullingActionsDeclareConfirm(t *testing.T) {
	for _, tool := range []struct {
		name    string
		actions []string
	}{
		{"service", []string{"preset_install", "update", "migrate", "rollback", "reinstall"}},
		{"runtime", []string{"ext_add"}},
	} {
		var schema mcpSchema
		for _, def := range toolList() {
			if def.Name == tool.name {
				schema = def.InputSchema
			}
		}
		if _, ok := schema.Properties["confirm"]; !ok {
			t.Errorf("%s takes no confirm argument", tool.name)
		}
		for _, action := range tool.actions {
			if _, ok := groupDispatch[tool.name][action]; !ok {
				t.Errorf("%s/%s is not a registered action (test list is stale)", tool.name, action)
			}
		}
	}
}
