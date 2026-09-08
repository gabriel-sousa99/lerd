package imagepull

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestParseRef(t *testing.T) {
	cases := []struct {
		ref             string
		host, repo, tag string
		ok              bool
	}{
		{"redis:7-alpine", "registry-1.docker.io", "library/redis", "7-alpine", true},
		{"mysql", "registry-1.docker.io", "library/mysql", "latest", true},
		{"postgis/postgis:16-3.5-alpine", "registry-1.docker.io", "postgis/postgis", "16-3.5-alpine", true},
		{"docker.io/library/nginx:alpine", "registry-1.docker.io", "library/nginx", "alpine", true},
		{"ghcr.io/lerd-env/php-base:84-abc123def456", "ghcr.io", "lerd-env/php-base", "84-abc123def456", true},
		{"quay.io/foo/bar@sha256:deadbeef", "quay.io", "foo/bar", "sha256:deadbeef", true},
		{"localhost/lerd-frankenphp84:local", "", "", "", false},
		{"lerd-php84-fpm:local", "", "", "", false},
		{"", "", "", "", false},
	}
	for _, tc := range cases {
		host, repo, tag, ok := parseRef(tc.ref)
		if ok != tc.ok || host != tc.host || repo != tc.repo || tag != tc.tag {
			t.Errorf("parseRef(%q) = %q %q %q %v, want %q %q %q %v",
				tc.ref, host, repo, tag, ok, tc.host, tc.repo, tc.tag, tc.ok)
		}
	}
}

// registryStub serves a multi-arch index that points at a platform manifest,
// behind the anonymous bearer-token challenge a public registry sends.
func registryStub(t *testing.T, requireToken bool) *httptest.Server {
	t.Helper()
	const digest = "sha256:child"
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			fmt.Fprint(w, `{"token":"abc"}`)
			return
		}
		if requireToken && r.Header.Get("Authorization") != "Bearer abc" {
			w.Header().Set("Www-Authenticate",
				`Bearer realm="`+srv.URL+`/token",service="reg",scope="repository:library/redis:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v2/library/redis/manifests/7-alpine":
			fmt.Fprintf(w, `{"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[
				{"digest":"sha256:other","platform":{"architecture":"ppc64le","os":"linux"}},
				{"digest":%q,"platform":{"architecture":%q,"os":"linux"}}]}`, digest, runtime.GOARCH)
		case "/v2/library/redis/manifests/" + digest:
			fmt.Fprint(w, `{"config":{"size":1000},"layers":[{"size":2000},{"size":3000}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSizeFromSumsPlatformManifest(t *testing.T) {
	srv := registryStub(t, false)
	got, ok := sizeFrom(srv.URL, "library/redis", "7-alpine", 0)
	if !ok || got != 6000 {
		t.Fatalf("sizeFrom = %d %v, want 6000 true", got, ok)
	}
}

func TestSizeFromAuthenticatesAnonymously(t *testing.T) {
	srv := registryStub(t, true)
	got, ok := sizeFrom(srv.URL, "library/redis", "7-alpine", 0)
	if !ok || got != 6000 {
		t.Fatalf("sizeFrom = %d %v, want 6000 true", got, ok)
	}
}

func TestSizeFromUnknownImage(t *testing.T) {
	srv := registryStub(t, false)
	if got, ok := sizeFrom(srv.URL, "library/nope", "latest", 0); ok || got != 0 {
		t.Fatalf("sizeFrom = %d %v, want 0 false", got, ok)
	}
}

func TestSizeSkipsLocalImages(t *testing.T) {
	if n, ok := Size("lerd-php84-fpm:local"); ok || n != 0 {
		t.Fatalf("Size(local) = %d %v, want 0 false", n, ok)
	}
}

func TestHuman(t *testing.T) {
	cases := map[int64]string{
		512:                    "512 B",
		2048:                   "2.0 KiB",
		18 * 1024 * 1024:       "18.0 MiB",
		3 * 1024 * 1024 * 1024: "3.0 GiB",
	}
	for n, want := range cases {
		if got := Human(n); got != want {
			t.Errorf("Human(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestReportDisclosesSizeAndReason(t *testing.T) {
	p := Plan{
		{Ref: "docker.io/library/redis:7-alpine", Reason: "redis is not installed", Bytes: 18 * 1024 * 1024},
		{Name: "PHP 8.4 image", Reason: "Containerfile changed", Build: true},
	}
	var buf bytes.Buffer
	p.Report(&buf)
	out := buf.String()
	for _, want := range []string{
		"lerd will download 2 images (~18.0 MiB total)",
		"pull    docker.io/library/redis:7-alpine",
		"~18.0 MiB  redis is not installed",
		"rebuild PHP 8.4 image",
		"size unknown  Containerfile changed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
}

// The disclosure is a lerd feedback block, not a bare printf: the download
// glyph heads it on the same left margin as the step lines around it, and the
// items are indented under it.
func TestReportUsesTheGlyphLayout(t *testing.T) {
	var buf bytes.Buffer
	Plan{{Ref: "redis:7-alpine", Reason: "redis is not installed", Bytes: 100}}.Report(&buf)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("report = %d lines, want a headline and one item:\n%s", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], " ↓ lerd will download 1 image") {
		t.Errorf("headline = %q, want the download glyph on the feedback margin", lines[0])
	}
	if !strings.HasPrefix(lines[1], "    pull ") {
		t.Errorf("item = %q, want it indented under the headline", lines[1])
	}
}

func TestReportDryRunSaysNothingWasDownloaded(t *testing.T) {
	SetDryRun(true)
	t.Cleanup(func() { SetDryRun(false) })
	var buf bytes.Buffer
	Plan{{Ref: "redis:7-alpine", Bytes: 100}}.Report(&buf)
	if !strings.Contains(buf.String(), "would download 1 image") ||
		!strings.Contains(buf.String(), "nothing was downloaded") {
		t.Fatalf("unexpected dry-run report:\n%s", buf.String())
	}
}

func TestReportEmptyPlanIsSilent(t *testing.T) {
	var buf bytes.Buffer
	Plan{}.Report(&buf)
	if buf.Len() != 0 {
		t.Fatalf("empty plan wrote %q", buf.String())
	}
}

func TestOfflineFromEnvAndFlag(t *testing.T) {
	if Offline() {
		t.Fatal("offline by default")
	}
	t.Setenv("LERD_OFFLINE", "1")
	if !Offline() {
		t.Fatal("LERD_OFFLINE=1 should enable offline")
	}
	t.Setenv("LERD_OFFLINE", "0")
	if Offline() {
		t.Fatal("LERD_OFFLINE=0 should not enable offline")
	}
	SetOffline(true)
	t.Cleanup(func() { SetOffline(false) })
	if !Offline() {
		t.Fatal("--no-pull should enable offline")
	}
}
