package ui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func postStartOnOpen(t *testing.T, enabled bool) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]bool{"enabled": enabled})
	req := httptest.NewRequest(http.MethodPost, "/api/settings/start-on-open", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleSettingsStartOnOpen(rec, req)
	return rec
}

func TestStartOnOpenPersistsAndIsReportedBySettings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	isolateLaunchAgents(t)

	if rec := postStartOnOpen(t, true); rec.Code != http.StatusOK {
		t.Fatalf("POST returned %d: %s", rec.Code, rec.Body.String())
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		t.Fatalf("loading config: %v", err)
	}
	if !cfg.Autostart.OnDashboardOpen {
		t.Error("config was not updated")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	handleSettings(rec, req)
	var resp SettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (%s)", err, rec.Body.String())
	}
	if !resp.StartOnDashboardOpen {
		t.Error("start_on_dashboard_open missing from GET /api/settings")
	}
}

func TestStartOnOpenTurnsBackOff(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	isolateLaunchAgents(t)

	postStartOnOpen(t, true)
	postStartOnOpen(t, false)

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Autostart.OnDashboardOpen {
		t.Error("the flag survived being turned off")
	}
}

func TestStartOnOpenRejectsNonPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/settings/start-on-open", nil)
	rec := httptest.NewRecorder()
	handleSettingsStartOnOpen(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET should be rejected, got %d", rec.Code)
	}
}

func TestStartOnOpenRejectsAnInvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/settings/start-on-open", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	handleSettingsStartOnOpen(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid body should be rejected, got %d", rec.Code)
	}
}
