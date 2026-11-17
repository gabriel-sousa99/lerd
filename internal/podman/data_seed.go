package podman

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// CopyImageData reads the image without starting its entrypoint or mounting
// the live data directory over the files it is meant to copy.
func CopyImageData(svc *config.CustomService, dest string) error {
	cmd := Cmd("create", "--pull=never", "--network=none", "--entrypoint=/bin/true", svc.Image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("creating image reader: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	id := strings.TrimSpace(string(out))
	defer func() { _ = Cmd("rm", "-v", id).Run() }()
	out, err = Cmd("cp", "--archive=false", id+":"+strings.TrimRight(svc.DataDir, "/")+"/.", dest).CombinedOutput()
	if err != nil {
		return fmt.Errorf("copying image data: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
