package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// OpenCode reads opencode.json and opencode.jsonc alike and its docs say to keep
// one format per directory, so a sibling .json beside an existing .jsonc leaves
// an entry that may never be read while the write reports success.
func TestResolveClientConfigPath_prefersAnExistingJSONC(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "opencode.json")
	jsoncPath := jsonPath + "c"

	if got := resolveClientConfigPath(jsonPath); got != jsonPath {
		t.Errorf("with no .jsonc: got %q, want %q", got, jsonPath)
	}
	if err := os.WriteFile(jsoncPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveClientConfigPath(jsoncPath[:len(jsoncPath)-1]); got != jsoncPath {
		t.Errorf("with a .jsonc: got %q, want %q", got, jsoncPath)
	}

	other := filepath.Join(dir, "mcp.json")
	if got := resolveClientConfigPath(other); got != other {
		t.Errorf("another client's config must not be redirected: got %q", got)
	}
}

func TestStripJSONComments(t *testing.T) {
	in := []byte(`{
  // the schema
  "$schema": "https://opencode.ai/config.json",
  /* block
     comment */
  "note": "a // inside a string stays",
  "theme": "tokyonight"
}`)
	var cfg map[string]any
	if err := json.Unmarshal(stripJSONComments(in), &cfg); err != nil {
		t.Fatalf("stripped JSONC did not parse: %v", err)
	}
	if cfg["note"] != "a // inside a string stays" {
		t.Errorf("note = %v, want the // preserved inside the string", cfg["note"])
	}
	if cfg["theme"] != "tokyonight" || cfg["$schema"] == nil {
		t.Errorf("cfg = %v, want every key to survive", cfg)
	}
}

// The merge has to reach the .jsonc a machine already uses, comments and all.
func TestWriteClientMCP_mergesIntoAnExistingJSONC(t *testing.T) {
	dir := t.TempDir()
	jsoncPath := filepath.Join(dir, "opencode.jsonc")
	if err := os.WriteFile(jsoncPath, []byte("{\n  // mine\n  \"theme\": \"tokyonight\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var oc aiClient
	for _, c := range aiClients {
		if c.Name == "opencode" {
			oc = c
		}
	}
	if err := writeClientMCP(filepath.Join(dir, "opencode.json"), oc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "opencode.json")); err == nil {
		t.Error("a sibling opencode.json was created beside the .jsonc")
	}
	data, err := os.ReadFile(jsoncPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("rewritten .jsonc did not parse: %v", err)
	}
	mcp, _ := cfg["mcp"].(map[string]any)
	if _, ok := mcp["lerd"]; !ok {
		t.Errorf("lerd entry missing from %v", cfg)
	}
	if cfg["theme"] != "tokyonight" {
		t.Errorf("existing keys were lost: %v", cfg)
	}
}
