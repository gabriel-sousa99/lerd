package podman

import (
	"os/exec"
	"reflect"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestCopyImageDataNeverMountsLiveDataAndAlwaysRemovesReader(t *testing.T) {
	for _, failCopy := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "copy failure"}[failCopy], func(t *testing.T) {
			original := execCommand
			t.Cleanup(func() { execCommand = original })
			var calls [][]string
			execCommand = func(_ string, args ...string) *exec.Cmd {
				calls = append(calls, append([]string(nil), args...))
				if args[0] == "create" {
					return exec.Command("printf", "reader-id\\n")
				}
				if args[0] == "cp" && failCopy {
					return exec.Command("false")
				}
				return exec.Command("true")
			}
			svc := &config.CustomService{Name: "seed-db", Image: "example.test/db:1", DataDir: "/data"}
			dest := t.TempDir()
			err := CopyImageData(svc, dest)
			if (err != nil) != failCopy {
				t.Fatalf("copy failure=%v: %v", failCopy, err)
			}
			want := [][]string{{"create", "--pull=never", "--network=none", "--entrypoint=/bin/true", svc.Image}, {"cp", "--archive=false", "reader-id:/data/.", dest}, {"rm", "-v", "reader-id"}}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("commands=%q, want %q", calls, want)
			}
		})
	}
}
