package podman

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The declared set is what drives a version's image content, so its fingerprint
// must move when the set does and stay put when it doesn't.
func TestCustomSetHash_tracksTheDeclaredSet(t *testing.T) {
	base := customSetHash("8.5", []string{"ssh2"}, map[string][]string{"ssh2": {"libssh2-dev"}}, []string{"chromium"})

	if got := customSetHash("8.5", []string{"ssh2"}, map[string][]string{"ssh2": {"libssh2-dev"}}, []string{"chromium"}); got != base {
		t.Error("the same declared set produced a different hash")
	}
	for _, tc := range []struct {
		name string
		hash string
	}{
		{"added extension", customSetHash("8.5", []string{"ssh2", "yaml"}, map[string][]string{"ssh2": {"libssh2-dev"}}, []string{"chromium"})},
		{"added package", customSetHash("8.5", []string{"ssh2"}, map[string][]string{"ssh2": {"libssh2-dev"}}, []string{"chromium", "jq"})},
		{"changed apk deps", customSetHash("8.5", []string{"ssh2"}, map[string][]string{"ssh2": {"krb5-dev"}}, []string{"chromium"})},
	} {
		if tc.hash == base {
			t.Errorf("%s did not drift the hash", tc.name)
		}
	}
}

// The set is a set: the order the user happened to add entries in must not
// rebuild every image.
func TestCustomSetHash_ignoresDeclarationOrder(t *testing.T) {
	a := customSetHash("8.5", []string{"ssh2", "yaml"}, nil, []string{"chromium", "jq"})
	b := customSetHash("8.5", []string{"yaml", "ssh2"}, nil, []string{"jq", "chromium"})
	if a != b {
		t.Error("declaration order changed the hash, which would rebuild images for nothing")
	}
}

// An empty declared set must fingerprint as empty. Images built before the
// label existed carry no label, which reads back as "": a user with no custom
// extensions must not have every image rebuilt on upgrade for nothing.
func TestCustomSetHash_emptySetIsEmpty(t *testing.T) {
	if got := customSetHash("8.5", nil, nil, nil); got != "" {
		t.Errorf("customSetHash(empty) = %q, want empty so unlabelled images still match", got)
	}
}

// The build skips a declared extension the image already ships, so it must not
// fingerprint the image either. An install that declared ftp before #1576 keeps
// the label it was built with, so dropping the name from the fingerprint is what
// makes that image read as stale once and rebuild onto the base image's own ftp.
func TestCustomSetHash_ignoresBundledExtensions(t *testing.T) {
	if got := customSetHash("8.5", []string{"ftp"}, map[string][]string{"ftp": {"openssl-dev"}}, nil); got != "" {
		t.Errorf("a set of nothing but bundled extensions must fingerprint as empty, got %q", got)
	}
	withBundled := customSetHash("8.5", []string{"ftp", "yaml"}, nil, nil)
	withoutBundled := customSetHash("8.5", []string{"yaml"}, nil, nil)
	if withBundled != withoutBundled {
		t.Error("a bundled extension alongside a custom one changed the fingerprint, so the image tracks something it never builds")
	}
	// The version gate still holds: 8.1's image has no ext/random to preserve.
	if customSetHash("8.1", []string{"random"}, nil, nil) == "" {
		t.Error("random is a real custom extension on 8.1 and must still fingerprint the image")
	}
}

// php -m prints display names: "Zend OPcache" is the module for the extension
// users install as "opcache". Matching raw lines made `php:ext add opcache`
// fail verification against an image that had loaded it perfectly well.
func TestPHPExtensionLoaded_canonicalisesDisplayNames(t *testing.T) {
	const modules = "[PHP Modules]\nCore\nZend OPcache\nSimpleXML\nPDO\nmongodb\n"

	for _, ext := range []string{"opcache", "simplexml", "pdo", "mongodb", "OPcache"} {
		if !phpExtensionLoaded(modules, ext) {
			t.Errorf("phpExtensionLoaded(%q) = false, want true", ext)
		}
	}
	for _, ext := range []string{"imagick", "", "xdebug"} {
		if phpExtensionLoaded(modules, ext) {
			t.Errorf("phpExtensionLoaded(%q) = true, want false", ext)
		}
	}
}

// The realised set is what the image actually has, which is never assumed from
// what was declared: mongodb does not build on 7.4 and the build swallows that
// failure by design, so only php -m and apk can be believed.
func TestRealisedSet_recordsOnlyWhatTheImageHas(t *testing.T) {
	const modules = "[PHP Modules]\nCore\nZend OPcache\nimagick\n"
	const apk = "jq\nbusybox\n"

	got := realisedSet(
		[]string{"imagick", "mongodb", "opcache"},
		[]string{"jq", "chromium"},
		modules, apk,
	)

	if want := []string{"imagick", "opcache"}; !reflect.DeepEqual(got.Extensions, want) {
		t.Errorf("realised extensions = %v, want %v (mongodb did not build)", got.Extensions, want)
	}
	if want := []string{"jq"}; !reflect.DeepEqual(got.Packages, want) {
		t.Errorf("realised packages = %v, want %v (chromium is not installed)", got.Packages, want)
	}
}

func TestRealisedSet_emptyDeclaredSetRealisesNothing(t *testing.T) {
	got := realisedSet(nil, nil, "[PHP Modules]\nCore\n", "busybox\n")
	if len(got.Extensions) != 0 || len(got.Packages) != 0 {
		t.Errorf("realisedSet(nothing declared) = %+v, want empty", got)
	}
}

// One declared set now reaches every version, including the Alpine 3.16 legacy
// images where a package may simply not exist. A single `apk add` of the whole
// list would fail the entire 7.4 build over one unavailable package, so each is
// installed tolerantly and the realised set records what actually landed. This
// mirrors what the custom extension block has always done with `|| true`.
func TestBuildCustomPackagesBlock_toleratesAPackageAVersionDoesNotHave(t *testing.T) {
	block := buildCustomPackagesBlock([]string{"chromium", "jq"})

	if block == "" {
		t.Fatal("no block generated for a non-empty package list")
	}
	if !strings.Contains(block, "|| true") {
		t.Errorf("package install is not tolerant, so one missing package fails the whole image:\n%s", block)
	}
	for _, pkg := range []string{"chromium", "jq"} {
		if !strings.Contains(block, pkg) {
			t.Errorf("block does not install %q:\n%s", pkg, block)
		}
	}
}

func TestBuildCustomPackagesBlock_emptyListGeneratesNothing(t *testing.T) {
	if got := buildCustomPackagesBlock(nil); got != "" {
		t.Errorf("buildCustomPackagesBlock(nil) = %q, want empty", got)
	}
}

// An existing image is only current if it was built from both the current
// recipe and the current declared set. Existence alone was the old test, and it
// is why an image could sit stale forever while the user waited for an
// extension they had already declared.
func TestFPMImageCurrent(t *testing.T) {
	origLabel := imageLabelFn
	origExec := execCommand
	t.Cleanup(func() { imageLabelFn = origLabel; execCommand = origExec })

	labels := map[string]string{}
	imageLabelFn = func(_, key string) string { return labels[key] }

	imageExists := true
	execCommand = func(name string, arg ...string) *exec.Cmd {
		if imageExists {
			return exec.Command("true")
		}
		return exec.Command("false")
	}

	labels[fpmContainerfileHashLabel] = "recipe1"
	labels[fpmCustomSetHashLabel] = "cust1"

	if !fpmImageCurrent("img", "recipe1", "cust1") {
		t.Error("an image matching both labels should be current")
	}
	if fpmImageCurrent("img", "recipe2", "cust1") {
		t.Error("a changed Containerfile must not count as current")
	}
	if fpmImageCurrent("img", "recipe1", "cust2") {
		t.Error("a newly declared extension must not count as current")
	}

	imageExists = false
	if fpmImageCurrent("img", "recipe1", "cust1") {
		t.Error("a missing image is never current")
	}

	// An image built before the custom-set label existed reports "". A user who
	// declares nothing hashes to "" too, so their images must not all rebuild.
	imageExists = true
	delete(labels, fpmCustomSetHashLabel)
	if !fpmImageCurrent("img", "recipe1", "") {
		t.Error("an unlabelled image must stay current when nothing is declared")
	}
	if fpmImageCurrent("img", "recipe1", "cust1") {
		t.Error("an unlabelled image must rebuild once something is declared")
	}
}

// The build stamps the current declared-set label on the image before this
// runs, so the image reads as fresh from that moment. Returning without a
// record then had MissingFromImage answer "nothing missing" for a version that
// was never measured, reporting the whole declared set as present — the #952
// false success reached through the absent record instead of the empty one.
func TestRecordRealisedSet_recordsAnEmptySetWhenTheImageCannotBeInspected(t *testing.T) {
	withTempXDG(t)
	origExec := execCommand
	t.Cleanup(func() { execCommand = origExec })
	execCommand = func(string, ...string) *exec.Cmd { return exec.Command("false") }

	RecordRealisedSet("8.0", []string{"mongodb"}, []string{"chromium"})

	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	missing := cfg.MissingFromImage("8.0", []string{"mongodb", "chromium"})
	if !reflect.DeepEqual(missing, []string{"mongodb", "chromium"}) {
		t.Errorf("MissingFromImage = %v, want [mongodb chromium]: an uninspectable build must not read as carrying the declared set", missing)
	}
	if cfg.GetRealised("8.0").Hash == "" {
		t.Error("the record must carry its hash, or it serializes away and reads back as no record at all")
	}
}

// The normal path still records exactly what the image reported.
func TestRecordRealisedSet_recordsWhatTheImageReports(t *testing.T) {
	withTempXDG(t)
	origExec := execCommand
	t.Cleanup(func() { execCommand = origExec })
	execCommand = func(_ string, arg ...string) *exec.Cmd {
		if len(arg) > 0 && arg[len(arg)-1] == "-m" {
			return exec.Command("printf", "%s\\n", "mongodb")
		}
		return exec.Command("printf", "%s\\n", "chromium")
	}

	RecordRealisedSet("8.4", []string{"mongodb", "imagick"}, []string{"chromium"})

	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	if missing := cfg.MissingFromImage("8.4", []string{"mongodb", "imagick", "chromium"}); !reflect.DeepEqual(missing, []string{"imagick"}) {
		t.Errorf("MissingFromImage = %v, want [imagick]", missing)
	}
}
