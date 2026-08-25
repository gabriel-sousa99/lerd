package linker

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// registered is the site the plan below would write, as the registry holds it.
func registered() config.Site {
	return config.Site{
		Name:        "app",
		Domains:     []string{"app.test", "admin.app.test"},
		Path:        "/home/u/app",
		PHPVersion:  "8.4",
		NodeVersion: "22",
		Secured:     true,
		Framework:   "laravel",
	}
}

func planFor(site config.Site) *Plan {
	return &Plan{Dir: site.Path, Site: site, Mode: ModeFPM}
}

func TestPlanMatchesRegistration(t *testing.T) {
	change := func(f func(*config.Site)) config.Site {
		s := registered()
		f(&s)
		return s
	}

	cases := []struct {
		name string
		plan config.Site
		want bool
	}{
		{"an unchanged plan writes what is already there", registered(), true},
		{"domain order is not a change", change(func(s *config.Site) {
			s.Domains = []string{"admin.app.test", "app.test"}
		}), true},
		{"a new domain is a change", change(func(s *config.Site) {
			s.Domains = append(s.Domains, "beta.app.test")
		}), false},
		{"a dropped domain is a change", change(func(s *config.Site) {
			s.Domains = []string{"app.test"}
		}), false},
		{"a different PHP version is a change", change(func(s *config.Site) { s.PHPVersion = "8.5" }), false},
		{"a different Node version is a change", change(func(s *config.Site) { s.NodeVersion = "24" }), false},
		{"HTTPS that is not on yet is a change", change(func(s *config.Site) { s.Secured = false }), false},
		{"a different framework is a change", change(func(s *config.Site) { s.Framework = "symfony" }), false},
		{"a runtime switch is a change", change(func(s *config.Site) { s.Runtime = "frankenphp" }), false},
		{"a custom container port is a change", change(func(s *config.Site) { s.ContainerPort = 3000 }), false},
		{"a host proxy port is a change", change(func(s *config.Site) { s.HostPort = 5173 }), false},
		{"a changed dev command is a change", change(func(s *config.Site) { s.HostCommand = "npm run dev" }), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := planFor(c.plan).MatchesRegistration(registered()); got != c.want {
				t.Errorf("MatchesRegistration() = %v, want %v", got, c.want)
			}
		})
	}
}

// The registry carries state no link decides — a paused site, a LAN share, a
// group, a worktree's pinned ports. None of it may read as a change, or the
// site would be re-linked on every setup for having been used.
func TestPlanMatchesRegistration_ignoresOperationalState(t *testing.T) {
	existing := registered()
	existing.Paused = true
	existing.Pinned = true
	existing.LANPort = 8080
	existing.Group = "app"
	existing.AppURL = "https://app.test"
	existing.WorktreeDevPorts = map[string]int{"feature": 5174}

	if !planFor(registered()).MatchesRegistration(existing) {
		t.Error("operational state read as a registration change")
	}
}

// Unlinking a site under a parked directory keeps its entry and marks it
// ignored, which hides it from every list and stops its workers. Re-linking must
// see work to do, or the link reports "already linked" and the site stays gone.
func TestPlanMatchesRegistration_ignoredEntryIsNotSettled(t *testing.T) {
	existing := registered()
	existing.Ignored = true

	if planFor(registered()).MatchesRegistration(existing) {
		t.Error("an unlinked parked site reported as already registered")
	}
}

// A plan that registers nothing (a worktree, or a directory the policy skips)
// never matches: there is no registration in it to compare.
func TestPlanMatchesRegistration_skippedPlanNeverMatches(t *testing.T) {
	p := planFor(registered())
	p.Skip = SkipWorktree
	if p.MatchesRegistration(registered()) {
		t.Error("a skipped plan reported as matching")
	}
}
