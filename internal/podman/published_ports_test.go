package podman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// writeQuadlet drops a container unit carrying the given PublishPort lines into
// the isolated quadlet dir, so the lookup reads the same file the port guard and
// the launchd translator do.
func writeQuadlet(t *testing.T, unit string, publish ...string) {
	t.Helper()
	dir := config.QuadletDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "[Container]\nImage=example\nContainerName=" + unit + "\n"
	for _, p := range publish {
		body += "PublishPort=" + p + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, unit+".container"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// A service publishing the Xdebug port answers the debugger's connect-back
// itself, so the owner has to be nameable, along with the container-internal
// port `lerd service port --container` needs to move the right mapping (#1555).
func TestPublishedPortOwner_NamesTheServiceAndMapping(t *testing.T) {
	setupConfigHome(t)
	writeQuadlet(t, "lerd-rustfs", "127.0.0.1:9002:9000", "[::1]:9002:9000", "127.0.0.1:9003:9001", "[::1]:9003:9001")

	owner, found := PublishedPortOwner(config.XdebugClientPort)
	if !found {
		t.Fatal("the service publishing the xdebug port was not found")
	}
	if owner.Unit != "lerd-rustfs" || owner.Service != "rustfs" {
		t.Errorf("owner = %+v, want unit lerd-rustfs / service rustfs", owner)
	}
	if owner.ContainerPort != 9001 {
		t.Errorf("container port = %d, want 9001: the secondary mapping is the one to move", owner.ContainerPort)
	}
}

// Nothing publishing the port is the ordinary case and must not be reported as
// a conflict.
func TestPublishedPortOwner_QuietWhenPortIsFree(t *testing.T) {
	setupConfigHome(t)
	writeQuadlet(t, "lerd-mysql", "127.0.0.1:3306:3306")

	if owner, found := PublishedPortOwner(config.XdebugClientPort); found {
		t.Errorf("reported %+v as publishing a free port", owner)
	}
}

// The ini the image reads and the port lerd reserves have to be the same number,
// or the reservation protects a port nothing debugs on.
func TestWriteXdebugIni_UsesTheReservedPort(t *testing.T) {
	setupConfigHome(t)
	if err := WriteXdebugIni("8.4", "debug", "yes"); err != nil {
		t.Fatalf("WriteXdebugIni: %v", err)
	}
	body, err := os.ReadFile(config.PHPConfFile("8.4"))
	if err != nil {
		t.Fatal(err)
	}
	want := "xdebug.client_port=9003"
	if !strings.Contains(string(body), want) {
		t.Errorf("ini does not carry %q:\n%s", want, body)
	}
	if config.XdebugClientPort != 9003 {
		t.Errorf("XdebugClientPort = %d, but the ini above pins 9003", config.XdebugClientPort)
	}
}
