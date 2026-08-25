package ui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/proxyops"
)

func TestProxyAPIListEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	req := httptest.NewRequest(http.MethodGet, "/api/proxies", nil)
	rr := httptest.NewRecorder()
	handleProxies(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty, got %d", len(list))
	}
}

func TestProxyAPICreateAndDelete(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	proxyops.StubForTests()
	defer proxyops.UnstubForTests()

	body, _ := json.Marshal(map[string]any{
		"domain":    "spa.localhost",
		"port":      9000,
		"no_secure": true,
		"path":      dir,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/proxies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleProxies(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status create: %d body=%s", rr.Code, rr.Body.String())
	}

	reg, _ := config.LoadProxies()
	if len(reg.Proxies) != 1 {
		t.Fatalf("expected 1 proxy")
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/proxies/spa", nil)
	delRR := httptest.NewRecorder()
	handleProxyAction(delRR, delReq)
	if delRR.Code != http.StatusNoContent {
		t.Fatalf("status delete: %d body=%s", delRR.Code, delRR.Body.String())
	}
}

func TestProxyAPIUpdatePartial(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	proxyops.StubForTests()
	defer proxyops.UnstubForTests()

	if _, err := proxyops.Add(proxyops.AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{"port": 9001})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/spa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleProxyAction(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int(got["upstream_port"].(float64)) != 9001 {
		t.Fatalf("upstream_port: got %v", got["upstream_port"])
	}

	reg, _ := config.LoadProxies()
	if reg.Proxies[0].UpstreamPort != 9001 {
		t.Fatalf("registry not updated: %d", reg.Proxies[0].UpstreamPort)
	}
}

func TestProxyAPIUpdateRejectsBadPort(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	proxyops.StubForTests()
	defer proxyops.UnstubForTests()

	if _, err := proxyops.Add(proxyops.AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"port": 70000})
	req := httptest.NewRequest(http.MethodPut, "/api/proxies/spa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleProxyAction(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: %d (expected 400)", rr.Code)
	}
}

func TestProxyAPICreatesAndUpdatesAdvancedSettings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	proxyops.StubForTests()
	defer proxyops.UnstubForTests()

	body, _ := json.Marshal(map[string]any{
		"domain": "spa.localhost", "aliases": []string{"admin.spa.localhost"},
		"port": 9443, "upstream_host": "127.0.0.1", "upstream_scheme": "https",
		"health_path": "/health", "timeout_seconds": 45, "no_secure": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/proxies", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handleProxies(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status create: %d body=%s", rr.Code, rr.Body.String())
	}

	var created proxyDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.UpstreamScheme != "https" || created.HealthPath != "/health" || created.TimeoutSeconds != 45 {
		t.Fatalf("advanced create fields missing: %+v", created)
	}
	if len(created.Domains) != 2 || created.Domains[1] != "admin.spa.localhost" {
		t.Fatalf("aliases missing: %+v", created.Domains)
	}

	newAliases := []string{"api.spa.localhost"}
	updateBody, _ := json.Marshal(map[string]any{
		"aliases": newAliases, "upstream_scheme": "http", "health_path": "/ready", "timeout_seconds": 10,
	})
	updateReq := httptest.NewRequest(http.MethodPut, "/api/proxies/spa", bytes.NewReader(updateBody))
	updateRR := httptest.NewRecorder()
	handleProxyAction(updateRR, updateReq)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("status update: %d body=%s", updateRR.Code, updateRR.Body.String())
	}

	var updated proxyDTO
	if err := json.Unmarshal(updateRR.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.UpstreamScheme != "http" || updated.HealthPath != "/ready" || updated.TimeoutSeconds != 10 {
		t.Fatalf("advanced update fields missing: %+v", updated)
	}
	if len(updated.Domains) != 2 || updated.Domains[1] != newAliases[0] {
		t.Fatalf("aliases not updated: %+v", updated.Domains)
	}
}

func TestProxyAPIActionRequiresPost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	proxyops.StubForTests()
	defer proxyops.UnstubForTests()

	if _, err := proxyops.Add(proxyops.AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/proxies/spa/pause", nil)
	rr := httptest.NewRecorder()
	handleProxyAction(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: %d (expected 405)", rr.Code)
	}
}
