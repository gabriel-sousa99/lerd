package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRouteJSONSnakeCaseRoundTrip(t *testing.T) {
	in := Route{Path: "/api", Site: "retencao-api", UpstreamPort: 8000, UpstreamHost: "127.0.0.1"}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	js := string(b)
	for _, want := range []string{`"path":"/api"`, `"site":"retencao-api"`, `"upstream_port":8000`, `"upstream_host":"127.0.0.1"`} {
		if !strings.Contains(js, want) {
			t.Errorf("json %s missing %s", js, want)
		}
	}
	// Decode snake_case back.
	var out Route
	if err := json.Unmarshal([]byte(`{"path":"/x","site":"s","upstream_port":9001,"upstream_host":"h"}`), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Path != "/x" || out.Site != "s" || out.UpstreamPort != 9001 || out.UpstreamHost != "h" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestRouteJSONOmitsEmpty(t *testing.T) {
	b, _ := json.Marshal(Route{Path: "/api", Site: "x"})
	js := string(b)
	if strings.Contains(js, "upstream_port") || strings.Contains(js, "upstream_host") {
		t.Errorf("empty port/host should be omitted: %s", js)
	}
}
