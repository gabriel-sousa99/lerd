package store

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

const testPackageYAML = `package: acme/electron
frameworks:
  - name: acme
    min: "11"
workers:
  native:
    label: Native
    command: php acme native:serve
    icon: nativemark
`

func packageServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/packages/acme-electron.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(testPackageYAML)) //nolint:errcheck
	})
	mux.HandleFunc("/packages/acme-impostor.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("package: acme/electron\n")) //nolint:errcheck
	})
	mux.HandleFunc("/workers/nativemark.svg", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M4 4h16v16H4z"/></svg>`)) //nolint:errcheck
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	return httptest.NewServer(mux)
}

func TestFetchPackage_cachesTheDefinitionAndItsWorkerIcons(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	srv := packageServer(t)
	defer srv.Close()

	pkg, err := testClient(t, srv).FetchPackage("acme/electron", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, has := pkg.Workers["native"]; !has {
		t.Error("worker missing from the parsed package")
	}
	if _, err := os.Stat(config.StorePackageFile("acme/electron", "")); err != nil {
		t.Errorf("definition was not cached: %v", err)
	}
	if _, ok := config.WorkerIcon("nativemark"); !ok {
		t.Error("worker mark was not cached alongside the definition")
	}
}

// The path a package is cached at comes from the name we asked for, so a file
// answering with another package's name is refused rather than written over it.
func TestFetchPackage_refusesADefinitionNamingAnotherPackage(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	srv := packageServer(t)
	defer srv.Close()

	if _, err := testClient(t, srv).FetchPackage("acme/impostor", ""); err == nil {
		t.Fatal("expected a mismatched package name to fail")
	}
	if _, err := os.Stat(config.StorePackageFile("acme/electron", "")); err == nil {
		t.Error("the impostor must not have been cached as acme/electron")
	}
}

func TestFetchPackage_rejectsANameThatIsNotAComposerPackage(t *testing.T) {
	srv := packageServer(t)
	defer srv.Close()

	if _, err := testClient(t, srv).FetchPackage("../../etc/passwd", ""); err == nil {
		t.Fatal("expected an invalid package name to fail")
	}
}
