package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProxyYAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	invalidateProxiesCache()

	original := Proxy{
		Name:         "gestao-clientes",
		Domains:      []string{"gestao-clientes.localhost"},
		UpstreamPort: 9000,
		UpstreamHost: "host.containers.internal",
		Path:         "/home/u/projetos/gestao-clientes-spa",
		Secured:      true,
		Managed:      true,
		NodeVersion:  "20",
		Cmd:          "npm run dev",
		AutoStart:    true,
	}

	reg := &ProxyRegistry{Proxies: []Proxy{original}}
	if err := SaveProxies(reg); err != nil {
		t.Fatalf("SaveProxies: %v", err)
	}

	invalidateProxiesCache()
	loaded, err := LoadProxies()
	if err != nil {
		t.Fatalf("LoadProxies: %v", err)
	}
	if len(loaded.Proxies) != 1 {
		t.Fatalf("want 1 proxy, got %d", len(loaded.Proxies))
	}
	got := loaded.Proxies[0]
	if got.Name != original.Name || got.UpstreamPort != 9000 || !got.Secured || !got.Managed {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.PrimaryDomain() != "gestao-clientes.localhost" {
		t.Fatalf("PrimaryDomain: %q", got.PrimaryDomain())
	}
}

func TestLoadProxiesMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	invalidateProxiesCache()
	reg, err := LoadProxies()
	if err != nil {
		t.Fatalf("LoadProxies: %v", err)
	}
	if len(reg.Proxies) != 0 {
		t.Fatalf("expected empty registry, got %d", len(reg.Proxies))
	}
}

func TestAddProxyDeduplicates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	invalidateProxiesCache()

	p := Proxy{Name: "gestao-clientes", Domains: []string{"gestao-clientes.localhost"}, UpstreamPort: 9000}
	if err := AddProxy(p); err != nil {
		t.Fatalf("AddProxy 1: %v", err)
	}
	updated := p
	updated.UpstreamPort = 9001
	if err := AddProxy(updated); err != nil {
		t.Fatalf("AddProxy 2: %v", err)
	}
	reg, _ := LoadProxies()
	if len(reg.Proxies) != 1 {
		t.Fatalf("expected dedup by name, got %d entries", len(reg.Proxies))
	}
	if reg.Proxies[0].UpstreamPort != 9001 {
		t.Fatalf("expected upsert to update port, got %d", reg.Proxies[0].UpstreamPort)
	}
}

func TestFindProxyByName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	invalidateProxiesCache()
	_ = AddProxy(Proxy{Name: "spa", Domains: []string{"spa.localhost"}, UpstreamPort: 9000})

	p, err := FindProxy("spa")
	if err != nil || p == nil || p.UpstreamPort != 9000 {
		t.Fatalf("FindProxy: %v / %+v", err, p)
	}

	if _, err := FindProxy("missing"); err == nil {
		t.Fatalf("expected error for missing proxy")
	}
}

func TestFindProxyByDomain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	invalidateProxiesCache()
	_ = AddProxy(Proxy{Name: "spa", Domains: []string{"spa.localhost"}, UpstreamPort: 9000})

	p, err := FindProxyByDomain("spa.localhost")
	if err != nil || p == nil || p.Name != "spa" {
		t.Fatalf("FindProxyByDomain: %v / %+v", err, p)
	}
}

func TestProxiesFilePath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	want := filepath.Join("/tmp/xdg", "lerd", "proxies.yaml")
	if got := ProxiesFile(); got != want {
		t.Fatalf("ProxiesFile: got %q want %q", got, want)
	}
	_ = os.Remove(want)
}
