package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageEstimateRejectsAPostRequest(t *testing.T) {
	rec := httptest.NewRecorder()
	handleImageEstimate(rec, httptest.NewRequest(http.MethodPost, "/api/image-estimate?service=redis&action=update", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// The estimate is always about a specific preset or service; without one there
// is nothing to size up and the dashboard should hear so rather than get a
// misleading empty answer.
func TestImageEstimateRequiresATarget(t *testing.T) {
	rec := httptest.NewRecorder()
	handleImageEstimate(rec, httptest.NewRequest(http.MethodGet, "/api/image-estimate", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// A PHP build has no service behind it, so the endpoint answers on the version
// alone and still names the image that build downloads.
func TestImageEstimateAnswersForAPHPVersion(t *testing.T) {
	rec := httptest.NewRecorder()
	handleImageEstimate(rec, httptest.NewRequest(http.MethodGet, "/api/image-estimate?php=8.4", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var out struct {
		Image string `json:"image"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if !strings.Contains(out.Image, "php84") {
		t.Errorf("image = %q, want the PHP 8.4 base", out.Image)
	}
}

func TestImageEstimateRejectsAnUnknownAction(t *testing.T) {
	rec := httptest.NewRecorder()
	handleImageEstimate(rec, httptest.NewRequest(http.MethodGet, "/api/image-estimate?service=redis&action=frobnicate", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
