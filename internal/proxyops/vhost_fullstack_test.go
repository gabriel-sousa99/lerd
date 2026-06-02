package proxyops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestRegenerateProxyVhost_Fullstack(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	orig := findSiteFn
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{
			Name: "retencao-api", Domains: []string{"retencao-api.localhost"},
			Path: "/home/u/retencao-api", PublicDir: "public", PHPVersion: "8.2",
		}, nil
	}
	defer func() { findSiteFn = orig }()

	p := config.Proxy{
		Name: "retencao", Domains: []string{"retencao.localhost"},
		UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "retencao-api"}},
	}
	if err := RegenerateProxyVhost(p); err != nil {
		t.Fatalf("RegenerateProxyVhost: %v", err)
	}

	confPath := filepath.Join(config.NginxConfD(), "retencao.localhost-ssl.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("lendo vhost gerado: %v", err)
	}
	for _, want := range []string{"location ^~ /api {", "location @site_retencao_api {", "lerd-php82-fpm"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("vhost não contém %q", want)
		}
	}
}

func TestRegenerateProxyVhost_SimpleUnchanged(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	p := config.Proxy{
		Name: "spa", Domains: []string{"spa.localhost"},
		UpstreamPort: 5173, Secured: true,
	}
	if err := RegenerateProxyVhost(p); err != nil {
		t.Fatalf("RegenerateProxyVhost: %v", err)
	}
	confPath := filepath.Join(config.NginxConfD(), "spa.localhost-ssl.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("lendo vhost gerado: %v", err)
	}
	// Proxy simples NÃO deve ter blocos fullstack.
	if strings.Contains(string(data), "location ^~") || strings.Contains(string(data), "@site_") {
		t.Errorf("proxy simples gerou config fullstack:\n%s", data)
	}
	// Deve ter o proxy_pass para a porta do dev server.
	if !strings.Contains(string(data), "proxy_pass http://host.containers.internal:5173;") {
		t.Errorf("proxy simples sem proxy_pass esperado:\n%s", data)
	}
}
