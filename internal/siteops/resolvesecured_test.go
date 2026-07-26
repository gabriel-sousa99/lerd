package siteops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestResolveSecured(t *testing.T) {
	dnsOn := &config.GlobalConfig{}
	dnsOn.DNS.Enabled = true
	dnsOff := &config.GlobalConfig{}
	dnsOff.DNS.Enabled = false

	securedProj := &config.ProjectConfig{Secured: true}
	plainProj := &config.ProjectConfig{Secured: false}

	cases := []struct {
		name   string
		relink bool
		proj   *config.ProjectConfig
		cfg    *config.GlobalConfig
		want   bool
	}{
		{"secured project, DNS managed", false, securedProj, dnsOn, true},
		// Upstream degrades these two to http; the fork serves HTTPS on
		// .localhost just as well, so the requested state is honoured.
		{"secured project, DNS disabled still secures", false, securedProj, dnsOff, true},
		{"re-link preserves prior secured when DNS disabled", true, plainProj, dnsOff, true},
		{"plain project, DNS managed", false, plainProj, dnsOn, false},
		{"no .lerd.yaml", false, nil, dnsOn, false},
		{"re-link preserves prior secured when DNS managed", true, plainProj, dnsOn, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveSecured(tc.relink, tc.proj, tc.cfg); got != tc.want {
				t.Errorf("ResolveSecured(%v, %+v, dns=%v) = %v, want %v",
					tc.relink, tc.proj, tc.cfg.DNS.Enabled, got, tc.want)
			}
		})
	}
}
