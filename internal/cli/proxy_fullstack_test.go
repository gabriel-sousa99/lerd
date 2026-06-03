package cli

import (
	"testing"
)

func TestBuildAPIRoutes_SiteWithDefaults(t *testing.T) {
	routes, err := buildAPIRoutes("retencao-api", 0, nil)
	if err != nil {
		t.Fatalf("buildAPIRoutes: %v", err)
	}
	// buildAPIRoutes deve mapear 1:1 cada path de defaultAPIPaths() para uma
	// Route com aquele Path e o site dado — verifica o mapeamento inteiro, não
	// só a contagem (a lista canônica é fixada em proxy_paths_test.go).
	want := defaultAPIPaths()
	if len(routes) != len(want) {
		t.Fatalf("len(routes) = %d, want %d: %+v", len(routes), len(want), routes)
	}
	for i, r := range routes {
		if r.Path != want[i] || r.Site != "retencao-api" || r.UpstreamPort != 0 {
			t.Errorf("routes[%d] = %+v, want Path=%q Site=retencao-api", i, r, want[i])
		}
	}
}

func TestBuildAPIRoutes_PortWithCustomPaths(t *testing.T) {
	routes, err := buildAPIRoutes("", 8000, []string{"/api"})
	if err != nil {
		t.Fatalf("buildAPIRoutes: %v", err)
	}
	if len(routes) != 1 || routes[0].UpstreamPort != 8000 || routes[0].Site != "" {
		t.Errorf("routes = %+v", routes)
	}
}

func TestBuildAPIRoutes_BothTargetsErr(t *testing.T) {
	if _, err := buildAPIRoutes("x", 8000, nil); err == nil {
		t.Error("esperava erro com site e porta juntos")
	}
}

func TestBuildAPIRoutes_NoneIsEmpty(t *testing.T) {
	routes, err := buildAPIRoutes("", 0, nil)
	if err != nil || routes != nil {
		t.Errorf("sem api target deve retornar nil,nil; got %+v err=%v", routes, err)
	}
}

func TestBuildAPIRoutes_PathsWithoutTargetErr(t *testing.T) {
	if _, err := buildAPIRoutes("", 0, []string{"/api"}); err == nil {
		t.Error("esperava erro quando --api-path é dado sem --api-site/--api-port")
	}
}
