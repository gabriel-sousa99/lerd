package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/reqstats"
)

func TestProxyStatusReportsHealthyUpstreamAndRuntimeArtifacts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("health path = %q, want /health", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	host, port := splitTestServerAddress(t, upstream.URL)
	p := config.Proxy{
		Name:         "spa",
		Domains:      []string{"spa.localhost"},
		UpstreamHost: host,
		UpstreamPort: port,
		HealthPath:   "/health",
	}
	if err := config.AddProxy(p); err != nil {
		t.Fatal(err)
	}

	vhost := filepath.Join(config.NginxConfD(), "spa.localhost.conf")
	if err := os.MkdirAll(filepath.Dir(vhost), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(vhost, []byte("server { server_name spa.localhost; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	originalNginxRunning := proxyNginxRunning
	proxyNginxRunning = func() bool { return true }
	t.Cleanup(func() { proxyNginxRunning = originalNginxRunning })

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/spa/status", nil)
	rr := httptest.NewRecorder()
	handleProxyAction(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	var got proxyStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.State != "healthy" || !got.UpstreamReachable || !got.NginxRunning || !got.VhostPresent {
		t.Fatalf("unexpected status: %+v", got)
	}
	if got.HTTPStatus != http.StatusNoContent {
		t.Fatalf("http status = %d, want %d", got.HTTPStatus, http.StatusNoContent)
	}
	if got.LatencyMillis < 0 || got.CheckedAt == "" {
		t.Fatalf("missing timing data: %+v", got)
	}
}

func TestProxyStatusSkipsProbeWhenPaused(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	if err := config.AddProxy(config.Proxy{
		Name: "spa", Domains: []string{"spa.localhost"}, UpstreamPort: 65534, Paused: true,
	}); err != nil {
		t.Fatal(err)
	}

	originalNginxRunning := proxyNginxRunning
	proxyNginxRunning = func() bool { return true }
	t.Cleanup(func() { proxyNginxRunning = originalNginxRunning })

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/spa/status", nil)
	rr := httptest.NewRecorder()
	handleProxyAction(rr, req)

	var got proxyStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.State != "paused" || got.UpstreamReachable {
		t.Fatalf("unexpected paused status: %+v", got)
	}
}

func TestProbeProxyUpstreamUsesPublishedDomainForSiteBackedBase(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("health path = %q, want /health", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	p := config.Proxy{
		Name:       "fullstack",
		Domains:    []string{u.Host},
		Site:       "api.localhost",
		HealthPath: "/health",
	}

	reachable, _, status, err := probeProxyUpstream(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reachable || status != http.StatusNoContent {
		t.Fatalf("reachable = %v, status = %d", reachable, status)
	}
}

func TestProxyGeneratedConfigIsReadOnlyAndResolvedFromRegistry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	if err := config.AddProxy(config.Proxy{
		Name: "spa", Domains: []string{"spa.localhost"}, UpstreamPort: 9000,
	}); err != nil {
		t.Fatal(err)
	}
	vhost := filepath.Join(config.NginxConfD(), "spa.localhost.conf")
	if err := os.MkdirAll(filepath.Dir(vhost), 0o755); err != nil {
		t.Fatal(err)
	}
	want := "server { server_name spa.localhost; }\n"
	if err := os.WriteFile(vhost, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/spa/config", nil)
	rr := httptest.NewRecorder()
	handleProxyAction(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	var got proxyConfigResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Content != want || got.Path != vhost {
		t.Fatalf("unexpected config response: %+v", got)
	}

	badReq := httptest.NewRequest(http.MethodGet, "/api/proxies/../config", nil)
	badRR := httptest.NewRecorder()
	handleProxyAction(badRR, badReq)
	if badRR.Code != http.StatusNotFound {
		t.Fatalf("traversal status = %d, want 404", badRR.Code)
	}
}

func TestProxyStatsReturnsNamespacedTrafficSnapshot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	if err := config.AddProxy(config.Proxy{
		Name: "spa", Domains: []string{"spa.localhost"}, UpstreamPort: 5173,
	}); err != nil {
		t.Fatal(err)
	}
	want := reqstats.SiteStats{
		Site: reqstats.ProxyKey("spa"), MedianMillis: 42, Samples: 8,
		Slow: []reqstats.RouteStat{}, UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := reqstats.SaveSnapshot([]reqstats.SiteStats{want}, config.RequestStatsFile()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/spa/stats", nil)
	rr := httptest.NewRecorder()
	handleProxyAction(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	var got reqstats.SiteStats
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Site != want.Site || got.Samples != want.Samples || got.MedianMillis != want.MedianMillis {
		t.Fatalf("stats = %+v, want %+v", got, want)
	}
}

func splitTestServerAddress(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return u.Hostname(), port
}
