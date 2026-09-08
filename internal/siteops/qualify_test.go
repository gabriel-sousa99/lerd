package siteops

import "testing"

// The help asks for a name without the TLD, but people type the domain they want
// to end up with. Appending to that turned shop.acme.test into shop.acme.test.test
// and put an unreachable alias in the site's .lerd.yaml.
func TestQualifyDomain(t *testing.T) {
	for _, tc := range []struct{ arg, want string }{
		{"shop", "shop.test"},
		{"shop.test", "shop.test"},
		{"shop.acme.test", "shop.acme.test"},
		{"SHOP.Test", "shop.test"},
		{"testapp", "testapp.test"},
		// Only a trailing TLD is the suffix; one in the middle is part of the name.
		{"test.shop", "test.shop.test"},
	} {
		if got := QualifyDomain(tc.arg, "test"); got != tc.want {
			t.Errorf("QualifyDomain(%q) = %q, want %q", tc.arg, got, tc.want)
		}
	}
}
