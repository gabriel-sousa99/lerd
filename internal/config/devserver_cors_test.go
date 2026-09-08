package config

import (
	"strings"
	"testing"
)

// A native shell serves the same app from 127.0.0.1 on a port it picks at boot,
// so its page has to be allowed to fetch what the site's page fetches, or the
// app loads with no scripts and one CORS error per file.
//
// Asserted against the shipped declaration rather than a copy of it: the wrapper
// in the cli tests is hand-written, so a change here would not reach it.
func TestDevServerWrapper_allowsLoopbackAlongsideTheSiteDomains(t *testing.T) {
	var vite *DevServerTool
	for i := range devServerTools {
		if devServerTools[i].Name == "vite" {
			vite = &devServerTools[i]
		}
	}
	if vite == nil {
		t.Fatal("no vite dev server is declared")
	}
	for _, want := range []string{
		`127\.0\.0\.1(:\d+)?`,
		`\[::1\](:\d+)?`,
		`localhost(:\d+)?`,
	} {
		if !strings.Contains(vite.Wrapper, want) {
			t.Errorf("loopback pattern %q missing from the generated config", want)
		}
	}
	// The site's own origins still lead: loopback is added to that list, not
	// swapped for it, or every other domain of the site loses its assets.
	if !strings.Contains(vite.Wrapper, "...%s") {
		t.Errorf("the declared origins are no longer spread into the allowlist:\n%s", vite.Wrapper)
	}
}
