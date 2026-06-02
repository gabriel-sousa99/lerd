package ui

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestToProxyDTO_Fullstack(t *testing.T) {
	p := config.Proxy{
		Name: "r", Domains: []string{"r.localhost"}, UpstreamPort: 9000,
		Routes: []config.Route{{Path: "/api", Site: "r-api"}},
	}
	dto := toProxyDTO(p)
	if !dto.Fullstack || len(dto.Routes) != 1 || dto.Routes[0].Site != "r-api" {
		t.Errorf("dto = %+v", dto)
	}
}

func TestToProxyDTO_SimpleNotFullstack(t *testing.T) {
	p := config.Proxy{Name: "s", Domains: []string{"s.localhost"}, UpstreamPort: 5173}
	dto := toProxyDTO(p)
	if dto.Fullstack || len(dto.Routes) != 0 {
		t.Errorf("simple proxy dto = %+v", dto)
	}
}
