package siteops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestEffectiveEnvDomain(t *testing.T) {
	origFind, origSync := findFullstackFn, syncEnvFn
	defer func() { findFullstackFn, syncEnvFn = origFind, origSync }()

	findFullstackFn = func(name string) (*config.Proxy, bool) {
		if name == "api" {
			return &config.Proxy{Domains: []string{"unified.localhost"}, Secured: true}, true
		}
		return nil, false
	}

	d, sec := effectiveEnvDomain(config.Site{Name: "api", Domains: []string{"api.localhost"}, Secured: false})
	if d != "unified.localhost" || !sec {
		t.Errorf("bound: got (%q,%v), want (unified.localhost,true)", d, sec)
	}
	d, sec = effectiveEnvDomain(config.Site{Name: "solo", Domains: []string{"solo.localhost"}, Secured: true})
	if d != "solo.localhost" || !sec {
		t.Errorf("unbound: got (%q,%v), want (solo.localhost,true)", d, sec)
	}
}
