package podman

import (
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// TestComposerPlatformNameRoundTrips checks both directions of the name folding:
// composer publishes OPcache as ext-zend-opcache (the module is "Zend OPcache",
// so extension_loaded("opcache") is false), while the image installs it as
// opcache. Both spellings must agree with BundledExtensions.
func TestComposerPlatformNameRoundTrips(t *testing.T) {
	cases := []struct{ bundled, platform string }{
		{"opcache", "zend-opcache"},
		{"redis", "redis"},
		{"pdo_mysql", "pdo_mysql"},
	}
	for _, c := range cases {
		if got := ComposerPlatformName(c.bundled); got != c.platform {
			t.Errorf("ComposerPlatformName(%q) = %q, want %q", c.bundled, got, c.platform)
		}
		if got := CanonicalExtension(c.platform); got != c.bundled {
			t.Errorf("CanonicalExtension(%q) = %q, want %q", c.platform, got, c.bundled)
		}
	}
}

func TestCanonicalExtensionIsCaseInsensitive(t *testing.T) {
	if got := CanonicalExtension("Zend-OPcache"); got != "opcache" {
		t.Errorf("CanonicalExtension(%q) = %q, want %q", "Zend-OPcache", got, "opcache")
	}
}

// TestBundledExtensionsUseInstallNames guards the mapping's premise: every alias
// key is a name BundledExtensions actually lists, so a rename in the list cannot
// silently orphan its composer counterpart.
func TestBundledExtensionsUseInstallNames(t *testing.T) {
	bundled := map[string]bool{}
	for _, e := range BundledExtensions("8.4") {
		bundled[e] = true
	}
	for install := range composerPlatformNames {
		if !bundled[install] {
			t.Errorf("composerPlatformNames maps %q, but BundledExtensions does not list it", install)
		}
	}
}

// TestBundledExtensionsCoverContainerfile checks one direction: every extension
// the FPM Containerfile installs or enables (docker-php-ext-install + the pecl
// docker-php-ext-enable names) is listed in BundledExtensions, so adding one to
// the image and forgetting the list is caught. This is NOT the #837 direction
// (BundledExtensions claiming a name nothing installs); that unverifiable core
// group is guarded by the php -m check in base-images.yml (#856).
func TestBundledExtensionsCoverContainerfile(t *testing.T) {
	cf, err := GetQuadletTemplate("lerd-php-fpm.Containerfile")
	if err != nil {
		t.Fatalf("reading FPM Containerfile: %v", err)
	}
	installed := fpmContainerfileExtensions(cf)
	if len(installed) < 10 {
		t.Fatalf("parsed only %d extensions, parser likely broke: %v", len(installed), installed)
	}
	bundled := map[string]bool{}
	for _, e := range BundledExtensions("8.4") {
		bundled[e] = true
	}
	for _, ext := range installed {
		if !bundled[ext] {
			t.Errorf("Containerfile installs %q but BundledExtensions omits it — park would never warn a project needs it", ext)
		}
	}
}

// A prerelease PHP outruns the PECL releases: they do not compile against it, so
// the image drops them and BundledExtensions must not promise what it never got.
func TestBundledExtensionsDropsPECLOnPrerelease(t *testing.T) {
	if len(config.PrereleasePHPVersions) == 0 {
		t.Skip("no prerelease version in the supported list")
	}
	v := config.PrereleasePHPVersions[0]
	got := strings.Join(BundledExtensions(v), " ")
	for _, ext := range []string{"igbinary", "pcov", "xdebug", "amqp", "memcached"} {
		if strings.Contains(" "+got+" ", " "+ext+" ") {
			t.Errorf("BundledExtensions(%q) advertises %q, which does not build on a prerelease", v, ext)
		}
	}
	// oci8 is in the kept list on purpose: it is the reason this fork exists and
	// it does build on the prerelease, so losing it has to fail here.
	for _, ext := range []string{"curl", "intl", "opcache", "pdo_mysql", "redis", "imagick", "mongodb", "oci8"} {
		if !strings.Contains(" "+got+" ", " "+ext+" ") {
			t.Errorf("BundledExtensions(%q) dropped %q, which the image does build", v, ext)
		}
	}
	stable := strings.Join(BundledExtensions("8.5"), " ")
	for _, ext := range []string{"xdebug", "amqp", "memcached"} {
		if !strings.Contains(" "+stable+" ", " "+ext+" ") {
			t.Errorf("BundledExtensions(8.5) lost %q; the prerelease rule leaked into a released version", ext)
		}
	}
}

// A declared extension the image already ships must be dropped: rebuilding it
// generically on top of the base image loses the configure flags the base build
// gave it, which is how ftp lost FTPS support again after #1583 (#1576).
func TestWithoutBundled(t *testing.T) {
	got := WithoutBundled("8.5", []string{"ftp", "yaml", "Zend-OPcache", "ssh2"})
	want := []string{"yaml", "ssh2"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("WithoutBundled = %v, want %v", got, want)
	}
}

// Version-gated names are not bundled everywhere, so a version whose image
// cannot build one must still be allowed to install it as a custom extension.
func TestWithoutBundled_KeepsVersionGatedNames(t *testing.T) {
	if got := WithoutBundled("8.1", []string{"random"}); len(got) != 1 {
		t.Errorf("WithoutBundled dropped random on 8.1, where the image does not ship it: %v", got)
	}
	if got := WithoutBundled("8.2", []string{"random"}); len(got) != 0 {
		t.Errorf("WithoutBundled kept random on 8.2, where the image ships it: %v", got)
	}
}
