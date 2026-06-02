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
