package origin

import (
	"strings"
	"testing"
)

// Every endpoint resolves to the org that actually publishes it — the fork for
// what this repository ships, upstream's lerd-env for the stores it does not
// fork — and never returns an empty list that would panic store.NewClient's
// urls[0]. The dead geodro org must not survive anywhere.
func TestAllEndpointsServeTheirPublisher(t *testing.T) {
	lists := map[string]struct {
		urls []string
		org  string
	}{
		"framework-store": {StoreBaseURLs(), "lerd-env"},
		"service-store":   {ServiceStoreBaseURLs(), "lerd-env"},
		"releases":        {ReleaseBaseURLs(), "gabriel-sousa99"},
		"downloads":       {ReleaseDownloadBases(), "gabriel-sousa99"},
		"api":             {ReleaseAPIBaseURLs(), "gabriel-sousa99"},
		"changelog":       {ChangelogURLs(), "gabriel-sousa99"},
		"baseimage":       {BaseImageRefs("85", "h"), "gabriel-sousa99"},
	}
	for name, tc := range lists {
		if len(tc.urls) == 0 {
			t.Fatalf("%s: empty base list", name)
		}
		if !strings.Contains(tc.urls[0], tc.org) {
			t.Errorf("%s: primary %q is not the %s location", name, tc.urls[0], tc.org)
		}
		for _, u := range tc.urls {
			if strings.Contains(u, "geodro") {
				t.Errorf("%s: must not reference geodro, got %q", name, u)
			}
		}
	}
}

func TestBaseImageRefFormat(t *testing.T) {
	refs := BaseImageRefs("84", "abc")
	if len(refs) != 1 || refs[0] != "ghcr.io/gabriel-sousa99/lerd-php84-fpm-base:abc" {
		t.Errorf("base ref = %v, want [ghcr.io/gabriel-sousa99/lerd-php84-fpm-base:abc]", refs)
	}
}

func TestBaseImageRegistryOverride(t *testing.T) {
	t.Setenv("LERD_BASE_IMAGE_REGISTRY", "registry.example/mirror")
	refs := BaseImageRefs("85", "h")
	if len(refs) != 1 || refs[0] != "registry.example/mirror/lerd-php85-fpm-base:h" {
		t.Errorf("override base ref = %v", refs)
	}
}

func TestStoreEnvOverrideReplacesList(t *testing.T) {
	t.Setenv("LERD_STORE_BASE_URL", "https://store.example/a, https://store.example/b")
	got := StoreBaseURLs()
	want := []string{"https://store.example/a", "https://store.example/b"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("override list = %v, want %v", got, want)
	}
}

func TestServiceStoreEnvOverride(t *testing.T) {
	t.Setenv("LERD_SERVICES_BASE_URL", "https://svc.example/a, https://svc.example/b")
	got := ServiceStoreBaseURLs()
	want := []string{"https://svc.example/a", "https://svc.example/b"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("service override list = %v, want %v", got, want)
	}
}

// A malformed override (only commas/whitespace) must be ignored and fall back to
// the default, never an empty list that would panic store.NewClient's urls[0].
func TestEnvOverrideIgnoredWhenEmpty(t *testing.T) {
	t.Setenv("LERD_STORE_BASE_URL", " , , ")
	got := StoreBaseURLs()
	if len(got) == 0 || !strings.Contains(got[0], "lerd-env") {
		t.Fatalf("empty override must fall back to the lerd-env framework store, got %v", got)
	}
}
