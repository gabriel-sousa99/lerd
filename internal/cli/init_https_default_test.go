package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// A new project answers "Enable HTTPS?" with Yes, so a wizard the user presses
// Enter through lands on https rather than leaving a site on http nobody chose.
// Anything already configured keeps what it has.
func TestResolveSecuredDefault(t *testing.T) {
	dnsOn := &config.GlobalConfig{}
	dnsOn.DNS.Enabled = true
	dnsOff := &config.GlobalConfig{}
	dnsOff.DNS.Enabled = false

	cases := []struct {
		name          string
		defaults      *config.ProjectConfig
		cfg           *config.GlobalConfig
		wantSecured   bool
		wantAvailable bool
	}{
		{"a new project starts on HTTPS", &config.ProjectConfig{}, dnsOn, true, true},
		{"no config at all is a new project", nil, dnsOn, true, true},
		{"a project asking for HTTPS keeps it", &config.ProjectConfig{PHPVersion: "8.4", Secured: true}, dnsOn, true, true},
		{"a configured project on http stays on http", &config.ProjectConfig{PHPVersion: "8.4"}, dnsOn, false, true},
		{"a localhost install is never asked", &config.ProjectConfig{}, dnsOff, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			secured, available := resolveSecuredDefault(t.TempDir(), c.defaults, c.cfg)
			if secured != c.wantSecured || available != c.wantAvailable {
				t.Errorf("resolveSecuredDefault = (%v, %v), want (%v, %v)",
					secured, available, c.wantSecured, c.wantAvailable)
			}
		})
	}
}

// A site the user secured by hand outranks a .lerd.yaml that never recorded it,
// so re-running the wizard does not offer to drop the HTTPS it is serving.
func TestResolveSecuredDefault_followsTheRegisteredSite(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	dir := t.TempDir()

	sites := "sites:\n  - name: app\n    path: " + dir + "\n    secured: true\n"
	sitesPath := filepath.Join(data, "lerd", "sites.yaml")
	if err := os.MkdirAll(filepath.Dir(sitesPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sitesPath, []byte(sites), 0o644); err != nil {
		t.Fatal(err)
	}

	dnsOn := &config.GlobalConfig{}
	dnsOn.DNS.Enabled = true
	secured, available := resolveSecuredDefault(dir, &config.ProjectConfig{PHPVersion: "8.4"}, dnsOn)
	if !secured || !available {
		t.Errorf("resolveSecuredDefault = (%v, %v), want (true, true) for a secured site", secured, available)
	}
}
