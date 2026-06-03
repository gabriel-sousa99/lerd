package cli

import "testing"

func TestDefaultAPIPaths_IncludesSSOAndConventions(t *testing.T) {
	got := defaultAPIPaths()
	want := []string{
		"/api", "/sanctum", "/broadcasting", "/storage",
		"/redirect", "/authenticate", "/login", "/logout", "/up",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d paths %v, want %d %v", len(got), got, len(want), want)
	}
	set := map[string]bool{}
	for _, p := range got {
		set[p] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("default paths não contém %q", w)
		}
	}
}
