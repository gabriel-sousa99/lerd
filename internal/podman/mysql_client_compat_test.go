package podman

import (
	"strings"
	"testing"
)

// MySQL 8.4+ authenticates root with caching_sha2_password, a plugin Alpine's
// mysql-client (the MariaDB client) only gains from mariadb-connector-c.
// Without it every `mysql` / `mysqldump` the framework shells out to fails.
func TestPHPImageShipsCachingSHA2Plugin(t *testing.T) {
	tmpl, err := GetQuadletTemplate("lerd-php-fpm.Containerfile")
	if err != nil {
		t.Fatalf("read Containerfile: %v", err)
	}
	runtime := tmpl[strings.Index(tmpl, "# ── Runtime stage"):]
	if !strings.Contains(runtime, "mariadb-connector-c") {
		t.Error("PHP-FPM runtime stage must install mariadb-connector-c")
	}
}

func TestMySQLClientCompatBlock(t *testing.T) {
	if !strings.Contains(mysqlClientCompatBlock, "mariadb-connector-c") {
		t.Error("fast-path layer must install mariadb-connector-c on older bases")
	}
	if !strings.Contains(mysqlClientCompatBlock, "caching_sha2_password.so") {
		t.Error("fast-path apk must be guarded so bases that ship the plugin stay offline-safe")
	}
	if !strings.Contains(mysqlClientCompatBlock, "lerd-no-ssl.cnf") {
		t.Error("fast-path layer must keep writing lerd-no-ssl.cnf")
	}
}
