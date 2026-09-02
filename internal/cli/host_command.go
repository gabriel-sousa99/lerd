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
//
// The project-supplied host command consent gate is deliberately not consulted:
// this declaration is store data, like the vite host worker, and a runtime that
// cannot live in a container does not work at all when it is refused.
func runDeclaredHostCommand(cwd string, argv []string, extraEnv []string) (int, bool, error) {
	fw := frameworkForDir(cwd)
	hc, ok := config.MatchHostCommand(fw, argv)
	if !ok {
		return 0, false, nil
	}
	bin := hc.Binary
	if !filepath.IsAbs(bin) {
		bin = filepath.Join(cwd, hc.Binary)
	}
	if _, err := os.Stat(bin); err != nil {
		return 0, true, errors.New(hostCommandMissingMsg(argv[0], hc))
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

// hostCommandMissingMsg reports what to do about a declared host binary the
// project has not installed yet. The command that installs it comes off the
// declaration rather than being written here, because two packages that both
// escape the container do not install the same way: nativephp/mobile installs
// through native:install-mobile precisely because the desktop package already
// owns native:install. A declaration naming none is not guessed at.
func hostCommandMissingMsg(argv0 string, hc config.HostCommand) string {
	msg := fmt.Sprintf("%s runs on the host and needs %s, which is not installed yet.", argv0, hc.Binary)
	if hc.InstallCommand == "" {
		return msg + "\nInstall the runtime it needs, then run it again."
	}
	return msg + "\nInstall the runtime first: lerd run " + hc.InstallCommand
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
// machine default. Empty is the version manager's own default alias, which is
// the right answer for a machine that has expressed no preference and keeps
// this off the platform-specific constants the worker units carry.
func hostCommandNodeVersion(cwd string) string {
	if v, err := nodeDet.DetectVersion(cwd); err == nil && v != "" {
		return v
	}
	if cfg, _ := config.LoadGlobal(); cfg != nil {
		return cfg.Node.DefaultVersion
	}
	return ""
}
