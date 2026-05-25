package proxyops

import (
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestGenerateManagedQuadlet(t *testing.T) {
	p := config.Proxy{
		Name:         "gestao",
		Domains:      []string{"gestao.localhost"},
		Path:         "/home/u/projetos/gestao-spa",
		UpstreamPort: 9000,
		Managed:      true,
		NodeVersion:  "20",
		Command:      "npm run dev",
		AutoStart:    true,
	}
	got := generateManagedQuadlet(p)

	mustContain := []string{
		"Description=Lerd proxy dev server (gestao)",
		"Image=docker.io/library/node:20-alpine",
		"ContainerName=lerd-proxy-gestao",
		"Network=host",
		"WorkingDir=/app",
		"Volume=/home/u/projetos/gestao-spa:/app:Z",
		`Exec=sh -lc "npm run dev"`,
		"Restart=on-failure",
		"WantedBy=default.target",
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("quadlet missing %q\n---\n%s", want, got)
		}
	}
}

func TestGenerateManagedQuadletDefaults(t *testing.T) {
	p := config.Proxy{
		Name:         "spa",
		Domains:      []string{"spa.localhost"},
		Path:         "/tmp/spa",
		UpstreamPort: 9000,
		Managed:      true,
		Command:      "npm run dev",
	}
	got := generateManagedQuadlet(p)
	if !strings.Contains(got, "Image=docker.io/library/node:20-alpine") {
		t.Errorf("expected default node:20-alpine:\n%s", got)
	}
	if strings.Contains(got, "WantedBy=default.target") {
		t.Errorf("AutoStart=false should not emit [Install]:\n%s", got)
	}
}
