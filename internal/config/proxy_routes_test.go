package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProxyRoutesYAMLRoundTrip(t *testing.T) {
	in := Proxy{
		Name:         "retencao",
		Domains:      []string{"retencao.localhost"},
		UpstreamPort: 9000,
		Site:         "spa-built",
		Secured:      true,
		Routes: []Route{
			{Path: "/api", Site: "retencao-api"},
			{Path: "/sanctum", Site: "retencao-api"},
			{Path: "/legacy", UpstreamPort: 8001, UpstreamHost: "127.0.0.1"},
		},
	}

	raw := proxyRegistryYAML{Proxies: []proxyYAML{in.toYAML()}}
	out, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back proxyRegistryYAML
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := back.Proxies[0].toProxy()

	if len(got.Routes) != 3 {
		t.Fatalf("routes = %d, want 3\nyaml:\n%s", len(got.Routes), out)
	}
	if got.Routes[0].Path != "/api" || got.Routes[0].Site != "retencao-api" {
		t.Errorf("route[0] = %+v", got.Routes[0])
	}
	if got.Routes[2].UpstreamPort != 8001 || got.Routes[2].UpstreamHost != "127.0.0.1" {
		t.Errorf("route[2] = %+v", got.Routes[2])
	}
	if got.Site != "spa-built" {
		t.Errorf("Site = %q, want spa-built", got.Site)
	}
	if !got.IsFullstack() {
		t.Error("IsFullstack() = false, want true")
	}
}

func TestSimpleProxyIsNotFullstack(t *testing.T) {
	p := Proxy{Name: "spa", Domains: []string{"spa.localhost"}, UpstreamPort: 5173}
	if p.IsFullstack() {
		t.Error("simple proxy reported as fullstack")
	}
}

func TestValidateProxyRoutes(t *testing.T) {
	cases := []struct {
		name    string
		routes  []Route
		wantErr bool
	}{
		{"ok", []Route{{Path: "/api", Site: "x"}}, false},
		{"ok port target", []Route{{Path: "/api", UpstreamPort: 8080}}, false},
		{"no leading slash", []Route{{Path: "api", Site: "x"}}, true},
		{"root path", []Route{{Path: "/", Site: "x"}}, true},
		{"duplicate", []Route{{Path: "/api", Site: "x"}, {Path: "/api", UpstreamPort: 9}}, true},
		{"no target", []Route{{Path: "/api"}}, true},
		{"both targets", []Route{{Path: "/api", Site: "x", UpstreamPort: 9}}, true},
		{"bad port", []Route{{Path: "/api", UpstreamPort: 70000}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProxyRoutes(tc.routes)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
