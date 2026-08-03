package podman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Enterprise Oracle is not reached by host:port/service. It is reached through a
// tnsnames.ora alias, and Autonomous Database through a downloaded wallet
// (cwallet.sso + sqlnet.ora). Instant Client looks for both under $TNS_ADMIN, so
// the image has to name that directory for an alias or a wallet to work at all.
func TestContainerfileSetsTNSAdmin(t *testing.T) {
	cf, err := GetQuadletTemplate("lerd-php-fpm.Containerfile")
	if err != nil {
		t.Fatalf("read Containerfile: %v", err)
	}
	if !strings.Contains(cf, "TNS_ADMIN=") {
		t.Error("Containerfile never sets TNS_ADMIN, so Instant Client cannot find tnsnames.ora or a wallet")
	}
	if !strings.Contains(cf, oracleTNSAdminContainerPath) {
		t.Errorf("Containerfile does not name %s as the TNS_ADMIN directory", oracleTNSAdminContainerPath)
	}
}

// A downloaded Autonomous wallet ships an sqlnet.ora whose DIRECTORY is
// "?/network/admin", and Oracle expands "?" to $ORACLE_HOME, which in this image
// is the Instant Client directory rather than the wallet's. Linking
// $ORACLE_HOME/network/admin at the real TNS_ADMIN makes an unedited wallet work,
// instead of every user having to discover and rewrite that line.
func TestContainerfileResolvesOracleHomeRelativeWalletPath(t *testing.T) {
	cf, err := GetQuadletTemplate("lerd-php-fpm.Containerfile")
	if err != nil {
		t.Fatalf("read Containerfile: %v", err)
	}
	if !strings.Contains(cf, "/opt/oracle/instantclient/network/admin") {
		t.Error("nothing resolves $ORACLE_HOME/network/admin, so a wallet's default sqlnet.ora DIRECTORY=\"?/network/admin\" points at an empty path")
	}
}

// The alias file and the wallet live on the host and must never be copied into an
// image, so they arrive as a read-only bind mount, the same way the user's SSH
// keys do.
func TestFPMQuadletMountsTNSAdminReadOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	content, err := renderFPMQuadletContent("8.4")
	if err != nil {
		t.Fatalf("renderFPMQuadletContent: %v", err)
	}
	want := ":" + oracleTNSAdminContainerPath + ":ro"
	if !strings.Contains(content, want) {
		t.Errorf("quadlet has no read-only mount ending in %q; got:\n%s", want, content)
	}
	// A writable mount would let a container rewrite the user's wallet.
	if strings.Contains(content, ":"+oracleTNSAdminContainerPath+":rw") {
		t.Error("TNS_ADMIN mount must be read-only")
	}
	// No placeholder may survive into a written unit.
	if strings.Contains(content, "{{.OracleTNSAdminDir}}") {
		t.Error("unsubstituted {{.OracleTNSAdminDir}} placeholder left in the quadlet")
	}
}

// The directory is lerd's own, so it is created rather than left to the user to
// guess: dropping a tnsnames.ora or an unzipped wallet into it is the whole
// setup step, and a bind mount of a missing source would otherwise fail.
func TestOracleTNSAdminHostDirIsCreated(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	dir := hostOracleTNSAdminDir()
	if dir == "/dev/null" {
		t.Fatal("host TNS_ADMIN dir fell back to /dev/null even though the config dir is writable")
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("host TNS_ADMIN dir %q was not created: %v", dir, err)
	}
	if !strings.HasPrefix(dir, cfg) {
		t.Errorf("host TNS_ADMIN dir %q is outside the lerd config dir %q", dir, cfg)
	}
}

// A wallet is a credential set, so the directory must not be world-readable.
func TestOracleTNSAdminHostDirIsNotWorldReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	info, err := os.Stat(hostOracleTNSAdminDir())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("TNS_ADMIN dir mode is %#o; a wallet directory must not be group- or world-readable", perm)
	}
}

// Whatever the user drops in must reach the container unchanged, so the mount
// source has to be the real directory rather than a copy under the data dir.
func TestOracleTNSAdminMountSourceIsTheHostDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	content, err := renderFPMQuadletContent("8.4")
	if err != nil {
		t.Fatalf("renderFPMQuadletContent: %v", err)
	}
	want := "Volume=" + hostOracleTNSAdminDir() + ":" + oracleTNSAdminContainerPath + ":ro"
	if !strings.Contains(content, want) {
		t.Errorf("expected mount line %q; got:\n%s", want, content)
	}
	_ = filepath.Separator
}
