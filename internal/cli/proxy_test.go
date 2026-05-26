package cli

import (
	"bytes"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/proxyops"
)

func TestProxyLsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	cmd := NewProxyCmd()
	cmd.SetArgs([]string{"ls"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	_ = buf.Bytes()
}

func TestProxyEditChangesPort(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	proxyops.StubForTests()
	defer proxyops.UnstubForTests()

	if _, err := proxyops.Add(proxyops.AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}

	cmd := NewProxyCmd()
	cmd.SetArgs([]string{"edit", "spa.localhost", "--port", "9001"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("edit: %v", err)
	}
	reg, _ := config.LoadProxies()
	if reg.Proxies[0].UpstreamPort != 9001 {
		t.Fatalf("port not updated: %d", reg.Proxies[0].UpstreamPort)
	}
}

func TestProxyEditNoFlagsNoOp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	proxyops.StubForTests()
	defer proxyops.UnstubForTests()

	if _, err := proxyops.Add(proxyops.AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}

	cmd := NewProxyCmd()
	cmd.SetArgs([]string{"edit", "spa.localhost"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("edit (no-op): %v", err)
	}
	reg, _ := config.LoadProxies()
	if reg.Proxies[0].UpstreamPort != 9000 {
		t.Fatalf("port mutated unexpectedly: %d", reg.Proxies[0].UpstreamPort)
	}
}

func TestProxyRmRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	proxyops.StubForTests()
	defer proxyops.UnstubForTests()

	_, err := proxyops.Add(proxyops.AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir})
	if err != nil {
		t.Fatal(err)
	}

	cmd := NewProxyCmd()
	cmd.SetArgs([]string{"rm", "spa.localhost"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("rm: %v", err)
	}
	reg, _ := config.LoadProxies()
	if len(reg.Proxies) != 0 {
		t.Fatalf("expected empty after rm")
	}
}
