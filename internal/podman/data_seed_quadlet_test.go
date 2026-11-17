package podman

import (
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestSeededServiceChecksDataBeforeBootWithoutRestartingOnBadData(t *testing.T) {
	svc := &config.CustomService{Name: "seed-db", Image: "example.test/db:1", DataDir: "/data", DataSeed: []string{"db/control"}}
	out := GenerateCustomQuadlet(svc)
	if !strings.Contains(out, "ExecCondition=%h/.local/bin/lerd service prepare-data seed-db\n") {
		t.Fatalf("missing boot guard: %s", out)
	}
	svc.DataSeed = nil
	if !strings.Contains(out, "TimeoutStartSec=300\n") {
		t.Fatal("initial image copy needs more than the default startup timeout")
	}
	if strings.Contains(GenerateCustomQuadlet(svc), "ExecCondition=") {
		t.Fatal("unseeded services must keep their existing startup")
	}
}
