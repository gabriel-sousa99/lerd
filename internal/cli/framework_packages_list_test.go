package cli

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestPackageDeclares(t *testing.T) {
	cases := []struct {
		name string
		in   config.StorePackageInfo
		want string
	}{
		{"workers and commands by name", config.StorePackageInfo{
			Cached: true, Workers: []string{"cron"}, Commands: []string{"cr", "uli"}, Setup: 3,
		}, "cron worker, cr, uli, 3 setup"},
		{"checks counted", config.StorePackageInfo{
			Cached: true, Workers: []string{"native"}, Doctor: 1,
		}, "native worker, 1 check"},
		{"nothing cached to describe", config.StorePackageInfo{}, "—"},
		{"cached but empty", config.StorePackageInfo{Cached: true}, "—"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := packageDeclares(tc.in); got != tc.want {
				t.Errorf("packageDeclares = %q, want %q", got, tc.want)
			}
		})
	}
}

// The file column says which definition answers, and admits when the store
// publishes one this machine has never pulled.
func TestPackageFileLabel(t *testing.T) {
	if got := packageFileLabel(config.StorePackageInfo{Cached: true}); got != "any version" {
		t.Errorf("unversioned = %q, want \"any version\"", got)
	}
	if got := packageFileLabel(config.StorePackageInfo{Cached: true, Version: "13"}); got != "@13" {
		t.Errorf("versioned = %q, want @13", got)
	}
	if got := packageFileLabel(config.StorePackageInfo{Version: "13"}); got != "not installed" {
		t.Errorf("uncached = %q, want \"not installed\"", got)
	}
}

// Run outside a project there is nothing to require, which is a different
// answer from a project that does not use the package.
func TestPackageUse(t *testing.T) {
	if got := packageUse(config.StorePackageInfo{Required: true}, "/tmp/site"); got != "yes" {
		t.Errorf("required = %q, want yes", got)
	}
	if got := packageUse(config.StorePackageInfo{}, "/tmp/site"); got != "no" {
		t.Errorf("not required = %q, want no", got)
	}
	if got := packageUse(config.StorePackageInfo{}, ""); got != "—" {
		t.Errorf("outside a project = %q, want an em dash", got)
	}
}
