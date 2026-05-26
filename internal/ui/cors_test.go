package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWithCORSAllowsMutatingMethods guards against a regression where
// /api/sites/{domain}/env (PUT) and /api/proxies/{name} (PUT) were
// blocked by a preflight that listed only GET/POST/DELETE/OPTIONS.
func TestWithCORSAllowsMutatingMethods(t *testing.T) {
	noopHandler := func(http.ResponseWriter, *http.Request) {}
	handler := withCORS(noopHandler)

	req := httptest.NewRequest(http.MethodOptions, "/api/anything", nil)
	req.Header.Set("Origin", "http://lerd.localhost")
	req.Header.Set("Access-Control-Request-Method", "PUT")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("preflight status: %d (expected 204)", rr.Code)
	}
	allow := rr.Header().Get("Access-Control-Allow-Methods")
	for _, want := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		if !strings.Contains(allow, want) {
			t.Errorf("Allow-Methods missing %s: %q", want, allow)
		}
	}
}
