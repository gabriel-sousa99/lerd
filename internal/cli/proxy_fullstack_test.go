package cli

import (
	"strings"
	"testing"
)

func TestDefaultAPIPaths(t *testing.T) {
	got := defaultAPIPaths()
	want := []string{"/api", "/sanctum", "/broadcasting", "/storage"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("defaultAPIPaths = %v, want %v", got, want)
	}
}

func TestBuildAPIRoutes_SiteWithDefaults(t *testing.T) {
	routes, err := buildAPIRoutes("retencao-api", 0, nil)
	if err != nil {
		t.Fatalf("buildAPIRoutes: %v", err)
	}
	if len(routes) != 4 || routes[0].Path != "/api" || routes[0].Site != "retencao-api" {
		t.Errorf("routes = %+v", routes)
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
