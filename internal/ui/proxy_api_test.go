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
