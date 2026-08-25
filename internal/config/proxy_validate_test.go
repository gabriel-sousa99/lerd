package config

import "testing"

func TestProxyValidate(t *testing.T) {
	cases := []struct {
		name    string
		p       Proxy
		wantErr bool
	}{
		{"simple ok", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000}, false},
		{"simple bad port", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 0}, true},
		{"simple with base site", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000, Site: "x"}, true},
		{"fullstack port base + site route", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000, Routes: []Route{{Path: "/api", Site: "x"}}}, false},
		{"fullstack site base", Proxy{Name: "a", Domains: []string{"a.localhost"}, Site: "spa", Routes: []Route{{Path: "/api", UpstreamPort: 8000}}}, false},
		{"fullstack base both", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000, Site: "spa", Routes: []Route{{Path: "/api", Site: "x"}}}, true},
		{"fullstack base neither", Proxy{Name: "a", Domains: []string{"a.localhost"}, Routes: []Route{{Path: "/api", Site: "x"}}}, true},
		{"fullstack bad route", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000, Routes: []Route{{Path: "api", Site: "x"}}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.p.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestProxyValidateAdvancedSettings(t *testing.T) {
	base := Proxy{Name: "app", Domains: []string{"app.localhost"}, UpstreamPort: 5173}
	cases := []struct {
		name    string
		mutate  func(*Proxy)
		wantErr bool
	}{
		{"defaults", func(*Proxy) {}, false},
		{"http", func(p *Proxy) { p.UpstreamScheme = "http" }, false},
		{"https", func(p *Proxy) { p.UpstreamScheme = "https" }, false},
		{"invalid scheme", func(p *Proxy) { p.UpstreamScheme = "ftp" }, true},
		{"health path", func(p *Proxy) { p.HealthPath = "/healthz" }, false},
		{"health path requires slash", func(p *Proxy) { p.HealthPath = "healthz" }, true},
		{"timeout lower bound", func(p *Proxy) { p.TimeoutSeconds = 1 }, false},
		{"timeout upper bound", func(p *Proxy) { p.TimeoutSeconds = 86400 }, false},
		{"negative timeout", func(p *Proxy) { p.TimeoutSeconds = -1 }, true},
		{"timeout too high", func(p *Proxy) { p.TimeoutSeconds = 86401 }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)
			if err := p.Validate(); (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestProxyAdvancedEffectiveDefaults(t *testing.T) {
	p := Proxy{}
	if got := p.EffectiveUpstreamScheme(); got != "http" {
		t.Fatalf("EffectiveUpstreamScheme() = %q, want http", got)
	}
	if got := p.EffectiveTimeoutSeconds(60); got != 60 {
		t.Fatalf("EffectiveTimeoutSeconds(60) = %d, want 60", got)
	}
	p.TimeoutSeconds = 120
	if got := p.EffectiveTimeoutSeconds(60); got != 120 {
		t.Fatalf("EffectiveTimeoutSeconds(60) = %d, want 120", got)
	}
}

func TestFindFullstackProxyForSite(t *testing.T) {
	reg := &ProxyRegistry{Proxies: []Proxy{
		{Name: "ret", Domains: []string{"ret.localhost"}, UpstreamPort: 9000, Routes: []Route{{Path: "/api", Site: "ret-api"}}},
		{Name: "blog", Domains: []string{"blog.localhost"}, Site: "blog-site"},
		{Name: "plain", Domains: []string{"plain.localhost"}, UpstreamPort: 3000},
	}}
	if p, ok := findFullstackProxyForSiteIn(reg, "ret-api"); !ok || p.Name != "ret" {
		t.Errorf("route site: got %v ok=%v", p, ok)
	}
	if p, ok := findFullstackProxyForSiteIn(reg, "blog-site"); !ok || p.Name != "blog" {
		t.Errorf("base site: got %v ok=%v", p, ok)
	}
	if _, ok := findFullstackProxyForSiteIn(reg, "nope"); ok {
		t.Error("expected not found")
	}
}
