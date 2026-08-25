package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestPresetResolve_CarriesStopTimeout pins that a preset's declared window
// survives version resolution. Preset embeds CustomService inline, so the field
// arrives for free, and this is the test that notices if that stops being true:
// the bundled database presets declare a longer window precisely so they are not
// SIGKILLed mid-checkpoint, and losing it in Resolve would be silent.
func TestPresetResolve_CarriesStopTimeout(t *testing.T) {
	var p Preset
	if err := yaml.Unmarshal([]byte(`
name: postgres
family: postgres
stop_timeout: 60
default_version: "16"
versions:
  - tag: "16"
    image: docker.io/postgis/postgis:16-3.5-alpine
    canonical: true
`), &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	svc, err := p.Resolve("16")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := svc.StopTimeoutSecs(); got != 60 {
		t.Errorf("resolved stop timeout = %d, want the preset's 60", got)
	}
}

// TestPresetResolve_UnversionedCarriesStopTimeout covers the other branch of
// Resolve, the one a preset with no versions block takes.
func TestPresetResolve_UnversionedCarriesStopTimeout(t *testing.T) {
	var p Preset
	if err := yaml.Unmarshal([]byte(`
name: mongo
image: docker.io/library/mongo:7
stop_timeout: 45
`), &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	svc, err := p.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := svc.StopTimeoutSecs(); got != 45 {
		t.Errorf("resolved stop timeout = %d, want the preset's 45", got)
	}
}

// TestPresetParse_IgnoresUnknownKeys is the backward-compatibility contract for
// the whole store. Definitions reach installed binaries within a day and there
// is no version gate, so a key added for a newer lerd lands in an older one's
// parser first. That has to be ignored rather than fail the parse, otherwise a
// single new field takes every older install's services down with it. Turning on
// KnownFields anywhere in this path would break that, and this test is what
// notices.
func TestPresetParse_IgnoresUnknownKeys(t *testing.T) {
	var p Preset
	if err := yaml.Unmarshal([]byte(`
name: postgres
image: docker.io/postgis/postgis:16-3.5-alpine
stop_timeout: 60
some_key_from_a_newer_lerd: whatever
nested_future_block:
  with: values
`), &p); err != nil {
		t.Fatalf("an unknown key from a newer store must not fail the parse: %v", err)
	}
	if p.Name != "postgres" {
		t.Errorf("Name = %q, want the known fields still populated", p.Name)
	}
}

// TestCustomServiceParse_IgnoresUnknownKeys is the same contract for a service
// definition rather than a preset.
func TestCustomServiceParse_IgnoresUnknownKeys(t *testing.T) {
	var svc CustomService
	if err := yaml.Unmarshal([]byte(`
name: futuredb
image: docker.io/library/postgres:17
stop_timeout: 60
some_key_from_a_newer_lerd: whatever
`), &svc); err != nil {
		t.Fatalf("an unknown key must not fail a service parse: %v", err)
	}
	if svc.StopTimeoutSecs() != 60 {
		t.Errorf("stop timeout = %d, want 60", svc.StopTimeoutSecs())
	}
}
