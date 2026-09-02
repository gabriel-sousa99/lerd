package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/geodro/lerd/internal/config"
	nodeDet "github.com/geodro/lerd/internal/node"
)

// runDeclaredHostCommand runs argv on the host when the project's framework
// says these arguments cannot work in the container, and reports whether it
// took the call. The case it exists for is a desktop runtime: native:run opens
// a window, and `php` on a lerd machine is the shim into the FPM container,
// where there is no Electron and no display. The declaration names the binary,
// so no framework or package is known here by name.
//
// A declared binary that is not there yet is an error rather than a fall back
// into the container, which is the failure the declaration exists to prevent.
func runDeclaredHostCommand(cwd string, argv []string, extraEnv []string) (int, bool, error) {
	fw := frameworkForDir(cwd)
	rel, ok := config.MatchHostCommand(fw, argv)
	if !ok {
		return 0, false, nil
	}
	bin := rel
	if !filepath.IsAbs(bin) {
		bin = filepath.Join(cwd, rel)
	}
	if _, err := os.Stat(bin); err != nil {
		return 0, true, fmt.Errorf("%s runs on the host and needs %s, which is not installed yet.\nInstall the runtime first: lerd run native:install", argv[0], rel)
	}
	// The command shells out to npm, so it needs the project's Node the same way
	// a host worker does. Unmanaged Node is left to the caller's PATH, which is
	// a real login shell here rather than a unit that inherits nothing.
	cmd := exec.Command(bin, argv...)
	if lerdManagesNode() {
		cmd = nodeDet.Active().Command(hostCommandNodeVersion(cwd), bin, argv)
	}
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode(), true, nil
		}
		return 0, true, err
	}
	return 0, true, nil
}

// frameworkForDir resolves the framework a directory's commands run under,
// package layer merged, or nil when the directory is not a project lerd knows.
func frameworkForDir(dir string) *config.Framework {
	if name, ok := config.DetectFrameworkForDir(dir); ok {
		if fw, ok := config.GetFrameworkForDir(name, dir); ok {
			return fw
		}
	}
	return nil
}

// hostCommandNodeVersion resolves the Node a re-routed command runs under, with
// the same fallbacks a host worker uses: the project's own version, then the
// machine default.
func hostCommandNodeVersion(cwd string) string {
	if v, err := nodeDet.DetectVersion(cwd); err == nil && v != "" {
		return v
	}
	if cfg, _ := config.LoadGlobal(); cfg != nil && cfg.Node.DefaultVersion != "" {
		return cfg.Node.DefaultVersion
	}
	return defaultNodeVersion
}
