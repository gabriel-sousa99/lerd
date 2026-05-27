package podman

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// WriteContainerUnitFn writes a container unit file for the given name and content.
// Defaults to writing a systemd quadlet (.container) file.
// Override this on macOS to write a launchd plist instead.
var WriteContainerUnitFn func(name, content string) error = WriteQuadlet

// DaemonReloadFn reloads the service manager after a unit file change.
// Defaults to systemctl --user daemon-reload.
// Override this on macOS with a no-op.
var DaemonReloadFn func() error = DaemonReload

// SkipQuadletUpToDateCheck disables the early-return optimisation in
// WriteFPMQuadlet that skips writing when the .container file is unchanged.
// Set to true on macOS where the unit file is a launchd plist, not a quadlet.
var SkipQuadletUpToDateCheck bool

// ExtraVolumePaths returns absolute paths that need to be bind-mounted into the
// PHP-FPM container because they are outside the user's home directory. It
// collects parked directories and linked site paths, deduplicates them, and
// returns only the top-level ancestors (so /var/www covers /var/www/app).
func ExtraVolumePaths() []string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}
	// Ensure home has a trailing slash for prefix matching.
	homePrefix := home
	if !strings.HasSuffix(homePrefix, "/") {
		homePrefix += "/"
	}

	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || p == home || strings.HasPrefix(p, homePrefix) {
			return
		}
		seen[p] = true
	}

	if cfg, err := config.LoadGlobal(); err == nil {
		for _, dir := range cfg.ParkedDirectories {
			add(dir)
		}
	}
	if reg, err := config.LoadSites(); err == nil {
		for _, site := range reg.Sites {
			add(site.Path)
		}
	}

	if len(seen) == 0 {
		return nil
	}

	// Collect unique paths and reduce to top-level ancestors.
	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	// Sort so shorter paths come first, then filter out children.
	sortPaths(paths)
	var result []string
	for _, p := range paths {
		covered := false
		for _, r := range result {
			rPrefix := r
			if !strings.HasSuffix(rPrefix, "/") {
				rPrefix += "/"
			}
			if strings.HasPrefix(p, rPrefix) || p == r {
				covered = true
				break
			}
		}
		if !covered {
			result = append(result, p)
		}
	}
	return result
}

// sortPaths sorts paths by length then lexicographically.
func sortPaths(paths []string) {
	for i := 1; i < len(paths); i++ {
		for j := i; j > 0; j-- {
			if len(paths[j]) < len(paths[j-1]) || (len(paths[j]) == len(paths[j-1]) && paths[j] < paths[j-1]) {
				paths[j], paths[j-1] = paths[j-1], paths[j]
			}
		}
	}
}

// mkcertPath returns the path to the mkcert binary managed by lerd.
func mkcertPath() string {
	return filepath.Join(config.BinDir(), "mkcert")
}

// mkcertCABlock copies the mkcert rootCA.pem into tmpDir and returns the
// Containerfile snippet that installs it into the Alpine trust store.
// Returns empty string if mkcert is not installed or the CA does not exist.
func mkcertCABlock(tmpDir string) string {
	out, err := exec.Command(mkcertPath(), "-CAROOT").Output()
	if err != nil {
		return ""
	}
	rootCA := filepath.Join(strings.TrimSpace(string(out)), "rootCA.pem")
	src, err := os.ReadFile(rootCA)
	if err != nil {
		return ""
	}
	dest := filepath.Join(tmpDir, "mkcert-ca.crt")
	if err := os.WriteFile(dest, src, 0644); err != nil {
		return ""
	}
	return "# Lerd mkcert CA — trust local .test HTTPS inside the container\n" +
		"COPY mkcert-ca.crt /usr/local/share/ca-certificates/mkcert-ca.crt\n" +
		"RUN update-ca-certificates\n"
}

// ContainerfileHash returns the SHA-256 hash of the embedded PHP-FPM Containerfile.
// This is used to detect when images need to be rebuilt after a lerd update.
func ContainerfileHash() (string, error) {
	tmpl, err := GetQuadletTemplate("lerd-php-fpm.Containerfile")
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(tmpl))
	return fmt.Sprintf("%x", sum), nil
}

// NeedsFPMRebuild returns true if the stored Containerfile hash differs from the
// current embedded Containerfile, meaning images should be rebuilt.
func NeedsFPMRebuild() bool {
	current, err := ContainerfileHash()
	if err != nil {
		return false
	}
	stored, err := os.ReadFile(config.PHPImageHashFile())
	if err != nil {
		// No stored hash yet — treat as needing rebuild only if images exist
		return false
	}
	return strings.TrimSpace(string(stored)) != current
}

// StoreFPMHash writes the current Containerfile hash to disk.
func StoreFPMHash() error {
	hash, err := ContainerfileHash()
	if err != nil {
		return err
	}
	return os.WriteFile(config.PHPImageHashFile(), []byte(hash), 0644)
}

// BuildFPMImage builds the lerd PHP-FPM image for the given version if it doesn't exist.
// When local is false, it attempts to pull a pre-built base image from ghcr.io first.
func BuildFPMImage(version string, local bool) error {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	return buildFPMImage(version, false, local, cfg.GetExtensions(version), cfg.AllExtApkDeps(), os.Stdout)
}

// BuildFPMImageTo builds the PHP-FPM image writing output to w.
// When local is false, it attempts to pull a pre-built base image from ghcr.io first.
func BuildFPMImageTo(version string, local bool, w io.Writer) error {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	return buildFPMImage(version, false, local, cfg.GetExtensions(version), cfg.AllExtApkDeps(), w)
}

// RebuildFPMImage force-removes and rebuilds the PHP-FPM image for the given version.
// When local is false, it attempts to pull a pre-built base image from ghcr.io first.
func RebuildFPMImage(version string, local bool) error {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	return buildFPMImage(version, true, local, cfg.GetExtensions(version), cfg.AllExtApkDeps(), os.Stdout)
}

// RebuildFPMImageTo force-rebuilds the PHP-FPM image writing output to w.
// When local is false, it attempts to pull a pre-built base image from ghcr.io first.
func RebuildFPMImageTo(version string, local bool, w io.Writer) error {
	cfg, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	return buildFPMImage(version, true, local, cfg.GetExtensions(version), cfg.AllExtApkDeps(), w)
}

// baseContainerfileHash returns a 12-character SHA-256 prefix of the Containerfile
// with user-specific sections stripped. This is used as the tag for pre-built base
// images on ghcr.io, so lerd knows exactly which image matches its embedded template.
func baseContainerfileHash() (string, error) {
	tmpl, err := GetQuadletTemplate("lerd-php-fpm.Containerfile")
	if err != nil {
		return "", err
	}
	base := strings.ReplaceAll(tmpl, "{{.CustomExtensions}}", "")
	base = strings.ReplaceAll(base, "{{.CustomExtensionsRuntime}}", "")
	base = strings.ReplaceAll(base, "{{.MkcertCA}}", "")
	sum := sha256.Sum256([]byte(base))
	return fmt.Sprintf("%x", sum)[:12], nil
}

// tryPullBaseImage attempts to pull the pre-built base image from ghcr.io.
// Returns the image reference on success, or "" if unavailable.
func tryPullBaseImage(version string, w io.Writer) string {
	hash, err := baseContainerfileHash()
	if err != nil {
		return ""
	}
	short := strings.ReplaceAll(version, ".", "")
	ref := fmt.Sprintf("ghcr.io/gabriel-sousa99/lerd-php%s-fpm-base:%s", short, hash)
	fmt.Fprintf(w, "  Pulling pre-built PHP %s base image...\n", version)

	// Use an empty auth file so the pull is always anonymous, regardless of
	// whether the user is logged into ghcr.io. A logged-in account with
	// expired or mismatched credentials would otherwise cause a 401 for this
	// public image and force a slow local build.
	tmpAuth, err := os.CreateTemp("", "lerd-auth-*.json")
	if err == nil {
		tmpAuth.WriteString("{}")
		tmpAuth.Close()
		defer os.Remove(tmpAuth.Name())
	}

	args := []string{"pull", "--policy=always"}
	if tmpAuth != nil {
		args = append(args, "--authfile="+tmpAuth.Name())
	}
	args = append(args, ref)

	cmd := exec.Command(PodmanBin(), args...)
	cmd.Stdout = w
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(w, "  Pre-built image unavailable, falling back to local build (may take a few minutes)...\n")
		return ""
	}
	return ref
}

func buildFPMImage(version string, force, local bool, customExts []string, extDeps map[string][]string, w io.Writer) error {
	short := strings.ReplaceAll(version, ".", "")
	imageName := "lerd-php" + short + "-fpm:local"

	if !force {
		// Skip if image already exists
		if exec.Command(PodmanBin(), "image", "exists", imageName).Run() == nil {
			return nil
		}
	}

	fmt.Fprintf(w, "\n  Building PHP %s image...\n", version)

	tmp, err := os.MkdirTemp("", "lerd-php-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	var containerfile string
	buildArgs := []string{"build", "-t", imageName}

	// Fast path: pull pre-built base and layer just mkcert CA + custom extensions on top.
	if !local {
		if baseRef := tryPullBaseImage(version, w); baseRef != "" {
			containerfile = "FROM " + baseRef + "\n" +
				"RUN mkdir -p /etc/my.cnf.d && printf '[client]\\nssl=0\\n' > /etc/my.cnf.d/lerd-no-ssl.cnf\n" +
				buildCustomExtBlock(customExts, extDeps) +
				mkcertCABlock(tmp)
			if force {
				buildArgs = append(buildArgs, "--no-cache")
			}
			goto build
		}
	}

	// Slow path: full local build from the embedded Containerfile template.
	{
		tmpl, tmplErr := GetQuadletTemplate("lerd-php-fpm.Containerfile")
		if tmplErr != nil {
			return tmplErr
		}
		containerfile = strings.ReplaceAll(tmpl, "{{.Version}}", version)
		containerfile = strings.ReplaceAll(containerfile, "{{.CustomExtensions}}", buildCustomExtBlock(customExts, extDeps))
		containerfile = strings.ReplaceAll(containerfile, "{{.CustomExtensionsRuntime}}", buildCustomExtRuntimeDeps(customExts, extDeps))
		containerfile = strings.ReplaceAll(containerfile, "{{.MkcertCA}}", mkcertCABlock(tmp))
		if force {
			// Bypass layer cache so changes are fully applied. The old image stays
			// tagged and the container keeps running until we restart the unit.
			buildArgs = append(buildArgs, "--no-cache")
		}
	}

build:
	cfPath := filepath.Join(tmp, "Containerfile")
	if err := os.WriteFile(cfPath, []byte(containerfile), 0644); err != nil {
		return err
	}

	buildArgs = append(buildArgs, "-f", cfPath, tmp)
	cmd := exec.Command(PodmanBin(), buildArgs...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("building PHP %s image: %w", version, err)
	}

	fmt.Fprintf(w, "  PHP %s image built successfully.\n", version)
	return nil
}

// extApkDeps maps a custom PHP extension to the Alpine packages its build needs.
// The standard bundle's -dev packages are already in the base image, so this only
// lists extensions whose build deps aren't there; without them PECL fails (e.g.
// imap's "U8T_CANONICAL is missing"). Users can add more via `lerd php:ext add
// --apk-deps`; the two sets are unioned. The "|| true" in the RUN block keeps a
// broken build from bricking later rebuilds, so VerifyExtensionLoaded checks the
// result afterward.
var extApkDeps = map[string][]string{
	"imap": {"imap-dev", "krb5-dev", "openssl-dev", "c-client"},
}

// validApkPkgName matches Alpine package names; used to reject anything that
// could break out of the `apk add` shell command in the generated Containerfile.
var validApkPkgName = regexp.MustCompile(`^[a-zA-Z0-9._+-]+$`)

// ParseApkDeps splits a space/comma/whitespace-separated package list and
// validates each name. Returns nil for empty input.
func ParseApkDeps(raw string) ([]string, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(fields) == 0 {
		return nil, nil
	}
	deps := make([]string, 0, len(fields))
	for _, f := range fields {
		if !validApkPkgName.MatchString(f) {
			return nil, fmt.Errorf("invalid Alpine package name %q", f)
		}
		deps = append(deps, f)
	}
	return deps, nil
}

// apkDepsForExt returns the union of the built-in and user-configured Alpine
// packages for ext, deduplicated, in a stable order.
func apkDepsForExt(ext string, userDeps map[string][]string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(pkgs []string) {
		for _, p := range pkgs {
			p = strings.TrimSpace(p)
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	add(extApkDeps[ext])
	add(userDeps[ext])
	return out
}

// buildCustomExtRuntimeDeps emits an apk RUN line that reinstalls the
// builder-stage deps in the runtime stage so compiled .so files can
// dlopen against those system libs. Empty when no custom exts have deps.
func buildCustomExtRuntimeDeps(exts []string, userDeps map[string][]string) string {
	seen := map[string]bool{}
	var deps []string
	for _, ext := range exts {
		for _, pkg := range apkDepsForExt(ext, userDeps) {
			if seen[pkg] {
				continue
			}
			seen[pkg] = true
			deps = append(deps, pkg)
		}
	}
	if len(deps) == 0 {
		return ""
	}
	return "RUN apk add --no-cache " + strings.Join(deps, " ") + " && rm -rf /var/cache/apk/*\n"
}

// buildCustomExtBlock generates Dockerfile RUN blocks for user-configured
// extensions, apk-adding any extra build deps (built-in map ∪ userDeps) first.
func buildCustomExtBlock(exts []string, userDeps map[string][]string) string {
	if len(exts) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("# User-configured extensions\n")
	for _, ext := range exts {
		prefix := ""
		if deps := apkDepsForExt(ext, userDeps); len(deps) > 0 {
			prefix = "apk add --no-cache " + strings.Join(deps, " ") + " && "
		}
		// `yes ''` feeds default answers to interactive PECL prompts (imap asks
		// for kerberos / c-client paths); harmless for extensions that don't ask.
		sb.WriteString(fmt.Sprintf(
			"RUN { %s(yes '' | pecl install %s && docker-php-ext-enable %s) || docker-php-ext-install %s || true; } \\\n    && rm -rf /tmp/pear /var/cache/apk/*\n",
			prefix, ext, ext, ext,
		))
	}
	return sb.String()
}

// phpExtensionLoaded reports whether ext appears in `php -m` output (case-insensitive).
func phpExtensionLoaded(moduleOutput, ext string) bool {
	want := strings.ToLower(strings.TrimSpace(ext))
	if want == "" {
		return false
	}
	for _, line := range strings.Split(moduleOutput, "\n") {
		if strings.ToLower(strings.TrimSpace(line)) == want {
			return true
		}
	}
	return false
}

// VerifyExtensionLoaded checks that the freshly built FPM image for the given
// version actually loads ext, by running `php -m` inside it. Returns an error if
// it isn't loaded (the PECL build failed and was swallowed by the "|| true" guard
// in the custom-extension RUN block).
func VerifyExtensionLoaded(version, ext string) error {
	short := strings.ReplaceAll(version, ".", "")
	imageName := "lerd-php" + short + "-fpm:local"
	out, err := exec.Command(PodmanBin(), "run", "--rm", imageName, "php", "-m").CombinedOutput()
	if err != nil {
		return fmt.Errorf("inspecting extensions in %s: %w\n%s", imageName, err, out)
	}
	if !phpExtensionLoaded(string(out), ext) {
		return fmt.Errorf("extension %q did not load in the rebuilt image (its build likely failed; check the extension name is correct, or pass --apk-deps with the Alpine packages it needs)", ext)
	}
	return nil
}

// validXdebugModes lists the xdebug.mode tokens accepted by NormaliseXdebugMode.
// Comma-separated combinations of these are allowed (e.g. "debug,coverage");
// "off" is only valid on its own.
var validXdebugModes = map[string]bool{
	"off":      true,
	"develop":  true,
	"coverage": true,
	"debug":    true,
	"gcstats":  true,
	"profile":  true,
	"trace":    true,
}

// NormaliseXdebugMode validates and canonicalises a user-supplied xdebug.mode
// value. Whitespace is trimmed, duplicates are dropped, and the result is a
// comma-separated string ready to be written into the ini file. An empty input
// returns "debug" so callers can use it as the default when enabling xdebug
// without an explicit mode.
func NormaliseXdebugMode(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "debug", nil
	}
	parts := strings.Split(raw, ",")
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !validXdebugModes[p] {
			return "", fmt.Errorf("invalid xdebug mode %q (accepted: debug, coverage, develop, profile, trace, gcstats, off)", p)
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	if len(out) == 0 {
		return "debug", nil
	}
	if len(out) > 1 && seen["off"] {
		return "", fmt.Errorf("xdebug mode %q cannot combine 'off' with other modes", raw)
	}
	return strings.Join(out, ","), nil
}

// WriteXdebugIni writes the per-version xdebug ini to the host config dir.
// The file is volume-mounted into the FPM container at /usr/local/etc/php/conf.d/99-xdebug.ini.
// An empty mode writes xdebug.mode=off (extension loaded but inactive); any other value
// is emitted as-is, so callers can pass "debug", "coverage", "debug,coverage", etc.
func WriteXdebugIni(version, mode string) error {
	path := config.PHPConfFile(version)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing stale xdebug ini directory: %w", err)
		}
	}
	if mode == "" {
		mode = "off"
	}
	// Oracle fork: start_with_request=trigger (not yes) so xdebug only fires
	// when the user explicitly opts in per-request via the XDEBUG_TRIGGER
	// cookie/header/query param. The upstream default of `yes` makes every
	// CLI command + every web request attempt a TCP connect to 9003 — when
	// no IDE is listening, that produces "Could not connect to debugging
	// client" spam on every artisan call. Users who want always-on debug
	// can flip it back via `lerd php:ini <v>`.
	content := fmt.Sprintf("[xdebug]\nxdebug.mode=%s\nxdebug.start_with_request=trigger\nxdebug.client_host=host.containers.internal\nxdebug.client_port=9003\n", mode)
	return os.WriteFile(path, []byte(content), 0644)
}

// ensureFPMHostsFile guarantees the bind-mount source for the FPM container's
// /etc/hosts is a regular file before podman starts the container. Three states
// are normalised here:
//
//  1. Path exists and is a directory (podman auto-created it on a previous
//     broken start, same race as the xdebug ini): remove it and fall through
//     to the missing-file branch.
//  2. Path is missing: try a real WriteContainerHosts; if that fails (e.g.
//     LoadSites errors), write a minimal static header so the mount still
//     succeeds and host.containers.internal resolves to something.
//  3. Path is already a regular file: no-op.
func ensureFPMHostsFile() error {
	hostsPath := config.ContainerHostsFile()
	info, err := os.Stat(hostsPath)
	if err == nil && info.IsDir() {
		if rmErr := os.Remove(hostsPath); rmErr != nil {
			return fmt.Errorf("removing stale hosts directory: %w", rmErr)
		}
		err = os.ErrNotExist
	}
	if !os.IsNotExist(err) {
		return nil
	}
	if writeErr := WriteContainerHosts(); writeErr == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(hostsPath), 0755); err != nil {
		return err
	}
	hostIP := DetectHostGatewayIP()
	return os.WriteFile(hostsPath, []byte(
		"127.0.0.1 localhost\n"+
			"::1 localhost\n"+
			hostIP+" host.containers.internal host.docker.internal\n",
	), 0644)
}

// EnsureXdebugIni creates the xdebug ini file for the given PHP version if it doesn't
// already exist as a regular file. This prevents Podman from auto-creating a directory
// at the bind-mount source path when the container starts before the file is written.
func EnsureXdebugIni(version string) error {
	path := config.PHPConfFile(version)
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		return nil // already a regular file
	}
	cfg, cfgErr := config.LoadGlobal()
	if cfgErr != nil {
		return cfgErr
	}
	return WriteXdebugIni(version, cfg.GetXdebugMode(version))
}

// WriteFPMQuadlet writes the systemd quadlet for a PHP-FPM version and reloads the
// systemd daemon if the content changed. It also ensures the xdebug and user ini files exist.
func WriteFPMQuadlet(version string) error {
	short := strings.ReplaceAll(version, ".", "")
	unitName := "lerd-php" + short + "-fpm"

	if err := EnsureUserIni(version); err != nil {
		return fmt.Errorf("creating user ini: %w", err)
	}
	if err := EnsureXdebugIni(version); err != nil {
		return fmt.Errorf("creating xdebug ini: %w", err)
	}
	if err := EnsureDumpAssets(); err != nil {
		return fmt.Errorf("ensuring dump assets: %w", err)
	}

	if err := ensureFPMHostsFile(); err != nil {
		return err
	}

	tmplContent, err := GetQuadletTemplate("lerd-php-fpm.container.tmpl")
	if err != nil {
		return err
	}
	content := strings.ReplaceAll(tmplContent, "{{.Version}}", version)
	content = strings.ReplaceAll(content, "{{.VersionShort}}", short)
	content = strings.ReplaceAll(content, "{{.XdebugIniPath}}", config.PHPConfFile(version))
	content = strings.ReplaceAll(content, "{{.UserIniPath}}", config.PHPUserIniFile(version))
	content = strings.ReplaceAll(content, "{{.DumpsDir}}", config.DumpsAssetsDir())
	content = strings.ReplaceAll(content, "{{.DumpsIniPath}}", config.DumpsIniFile())
	content = strings.ReplaceAll(content, "{{.HostNameLine}}", hostNameLine())
	content = strings.ReplaceAll(content, "{{.HostSSHDir}}", hostSSHDir())
	content = applyShellMounts(content, short)
	content = InjectExtraVolumes(content, ExtraVolumePaths())
	content = injectDevServerPorts(content, version)

	// Skip the write and daemon-reload if the quadlet is already up to date.
	// Unnecessary daemon-reloads cause Podman's quadlet generator to regenerate
	// all service files, which can briefly disrupt lerd-dns and cause
	// systemd-resolved to mark 127.0.0.1:5300 as failed (breaking .test resolution).
	// On macOS the unit file is a launchd plist (not a quadlet), so the check is skipped.
	if !SkipQuadletUpToDateCheck {
		existingPath := filepath.Join(config.QuadletDir(), unitName+".container")
		if existing, err := os.ReadFile(existingPath); err == nil && string(existing) == content {
			return nil
		}
	}

	changed, err := WriteQuadletDiff(unitName, content)
	if err != nil {
		return err
	}
	if err := DaemonReloadFn(); err != nil {
		return err
	}
	// If the container is already running with the OLD quadlet (e.g. user
	// reinstalled or re-linked a site on the same PHP version), the running
	// FPM picks up the new mounts only after a restart. Without this, the
	// next `podman exec -w /new/path ...` fails with `runc chdir failed:
	// no such file or directory` even though the quadlet on disk is correct.
	// Skip the restart for brand-new versions whose container hasn't been
	// started yet — the next `lerd start` will pick up the fresh config.
	if changed && ContainerRunningQuiet(unitName) {
		if err := RestartUnit(unitName); err != nil {
			return fmt.Errorf("restart %s after quadlet update: %w", unitName, err)
		}
	}
	return nil
}

// devServerPublishPorts returns the host PublishPort specs that expose a PHP
// container's dev-server ports — artisan serve (8000/8001) and websockets
// (6001) — so the host (and, under WSL, the Windows browser via localhost
// forwarding) can reach a server the user starts inside the container.
//
// PHP 5.6 is deliberately excluded: it is the legacy tier that commonly runs
// alongside a modern version, and two FPM containers cannot both publish the
// same host ports. Leaving 5.6 unbound lets it coexist with (say) 8.4 without
// a port collision.
//
// The 0.0.0.0 form is preserved as-is by BindForLAN (it only rewrites bare and
// loopback binds), so these dev ports stay reachable on every interface
// regardless of the global lan:expose toggle, without affecting the loopback
// data services.
func devServerPublishPorts(version string) []string {
	if version == "5.6" {
		return nil
	}
	return []string{
		"0.0.0.0:8000:8000",
		"0.0.0.0:8001:8001",
		"0.0.0.0:6001:6001",
	}
}

// injectDevServerPorts inserts the dev-server PublishPort lines into the
// [Container] section (immediately after the Network= line) of an FPM quadlet
// for the given PHP version. Returns content unchanged for versions that
// expose nothing (e.g. 5.6) or when no Network= anchor is present.
func injectDevServerPorts(content, version string) string {
	ports := devServerPublishPorts(version)
	if len(ports) == 0 {
		return content
	}
	portLines := make([]string, len(ports))
	for i, p := range ports {
		portLines[i] = "PublishPort=" + p
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "Network=") {
			continue
		}
		out := make([]string, 0, len(lines)+len(portLines))
		out = append(out, lines[:i+1]...)
		out = append(out, portLines...)
		out = append(out, lines[i+1:]...)
		return strings.Join(out, "\n")
	}
	return content
}

// RewriteFPMQuadlets regenerates the quadlet files for all installed PHP-FPM
// versions and the nginx quadlet. Call this when parked directories or site
// paths change so that extra volume mounts stay in sync.
func RewriteFPMQuadlets() error {
	extraPaths := ExtraVolumePaths()
	versions, _ := listInstalledPHPVersions()

	var changedUnits []string

	for _, v := range versions {
		short := strings.ReplaceAll(v, ".", "")
		unitName := "lerd-php" + short + "-fpm"

		tmplContent, tmplErr := GetQuadletTemplate("lerd-php-fpm.container.tmpl")
		if tmplErr != nil {
			continue
		}
		content := strings.ReplaceAll(tmplContent, "{{.Version}}", v)
		content = strings.ReplaceAll(content, "{{.VersionShort}}", short)
		content = strings.ReplaceAll(content, "{{.XdebugIniPath}}", config.PHPConfFile(v))
		content = strings.ReplaceAll(content, "{{.UserIniPath}}", config.PHPUserIniFile(v))
		content = strings.ReplaceAll(content, "{{.DumpsDir}}", config.DumpsAssetsDir())
		content = strings.ReplaceAll(content, "{{.DumpsIniPath}}", config.DumpsIniFile())
		content = strings.ReplaceAll(content, "{{.HostNameLine}}", hostNameLine())
		content = strings.ReplaceAll(content, "{{.HostSSHDir}}", hostSSHDir())
		content = applyShellMounts(content, short)
		content = InjectExtraVolumes(content, extraPaths)
		content = injectDevServerPorts(content, v)

		changed, writeErr := WriteQuadletDiff(unitName, content)
		if writeErr != nil {
			continue
		}
		if changed {
			changedUnits = append(changedUnits, unitName)
		}
	}

	// Also rewrite nginx quadlet with the same extra volumes.
	if nginxContent, err := GetQuadletTemplate("lerd-nginx.container"); err == nil {
		nginxContent = InjectExtraVolumes(nginxContent, extraPaths)
		if changed, err := WriteQuadletDiff("lerd-nginx", nginxContent); err == nil && changed {
			changedUnits = append(changedUnits, "lerd-nginx")
		}
	}

	if len(changedUnits) > 0 {
		var errs []error
		if err := DaemonReload(); err != nil {
			errs = append(errs, fmt.Errorf("daemon-reload after quadlet rewrite: %w", err))
		}
		for _, unit := range changedUnits {
			if err := RestartUnit(unit); err != nil {
				errs = append(errs, fmt.Errorf("restart %s: %w", unit, err))
			}
		}
		// Nginx may have restarted and received a new IP. Regenerate the
		// browser-testing hosts file so Selenium resolves .test domains to
		// the current nginx container address.
		if err := WriteContainerHosts(); err != nil {
			errs = append(errs, fmt.Errorf("write container hosts: %w", err))
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
	}
	return nil
}

// zshHistoryDir returns the per-PHP-version host directory that backs the
// container's /root/.zsh_state mount, creating it so the bind mount succeeds
// on first start. We deliberately do not mount any host shell config —
// see internal/podman/quadlets/lerd-php-fpm.Containerfile for the rationale.
func zshHistoryDir(versionShort string) string {
	dir := filepath.Join(config.DataDir(), "shell-state", "php-"+versionShort, "zsh")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// hostSSHDir returns the user's $HOME/.ssh path for the FPM quadlet's
// read-only mount into /root/.ssh — lets composer/git inside the container
// authenticate against remote repos with the user's host keys. Falls back
// to /dev/null when $HOME/.ssh doesn't exist so the mount line is still
// syntactically valid (podman tolerates a /dev/null source).
func hostSSHDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/dev/null"
	}
	candidate := filepath.Join(home, ".ssh")
	if info, statErr := os.Stat(candidate); statErr != nil || !info.IsDir() {
		return "/dev/null"
	}
	return candidate
}

// hostNameLine returns the `HostName=<host>` directive for the FPM quadlet so
// prompts inside the container read e.g. "root@laptop" instead of the
// auto-generated podman container id. Returns an empty string when the host
// hostname can't be read or contains characters podman would reject, so the
// placeholder line collapses cleanly.
func hostNameLine() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	h = strings.TrimSpace(h)
	if h == "" {
		return ""
	}
	for _, r := range h {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '.'
		if !ok {
			return ""
		}
	}
	return "HostName=" + h
}

// applyShellMounts substitutes shell-related template fields.
func applyShellMounts(content, versionShort string) string {
	return strings.ReplaceAll(content, "{{.ZshHistoryDir}}", zshHistoryDir(versionShort))
}

// listInstalledPHPVersions returns PHP versions that have a quadlet installed.
func listInstalledPHPVersions() ([]string, error) {
	dir := config.QuadletDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "lerd-php") || !strings.HasSuffix(name, "-fpm.container") {
			continue
		}
		// Extract version short from lerd-php84-fpm.container → "84"
		short := strings.TrimPrefix(name, "lerd-php")
		short = strings.TrimSuffix(short, "-fpm.container")
		if len(short) < 2 {
			continue
		}
		// Convert "84" → "8.4"
		version := string(short[0]) + "." + short[1:]
		versions = append(versions, version)
	}
	return versions, nil
}

// ephemeralPathPrefixes lists filesystem trees that are system-managed and
// short-lived — there is no reason to bind-mount them into FPM/nginx, and
// IDEs (PhpStorm, VSCode) drop temp .php files into /tmp with random names
// that, mounted blindly, cascade FPM restarts every time the IDE invokes
// `php` on a fresh path.
var ephemeralPathPrefixes = []string{
	"/tmp/", "/var/tmp/",
	"/run/", "/proc/", "/sys/", "/dev/",
}

// pathMountAttempts memoises recent EnsurePathMounted calls so a runaway
// caller (IDE running `php` repeatedly with rotating temp paths, broken
// shell loop) cannot keep rewriting the FPM quadlet and re-triggering
// RestartUnit at the cadence required to hit systemd's start rate-limit.
//
// The debounce window depends on outcome:
//   - successful restart  → pathMountSuccessDebounce (long, prevents cascade)
//   - failed restart       → pathMountFailureDebounce (short, allows retry
//     once the transient DBus/systemd hiccup clears)
//
// Without the failure path the user can hit a 60-second blackhole where a
// silently-failed restart leaves the container with stale mounts and every
// subsequent `lerd php` call early-returns without retrying.
var (
	pathMountAttemptsMu sync.Mutex
	pathMountAttempts   = map[string]pathMountStamp{}
)

type pathMountStamp struct {
	when     time.Time
	debounce time.Duration
}

const (
	pathMountSuccessDebounce = 60 * time.Second
	pathMountFailureDebounce = 5 * time.Second
)

// EnsurePathMounted checks whether the given path is accessible inside the
// PHP-FPM and nginx containers. If the path is outside $HOME and not already
// volume-mounted, the quadlets are updated and containers restarted
// transparently before returning.
func EnsurePathMounted(path, phpVersion string) {
	home, _ := os.UserHomeDir()
	if home == "" {
		return
	}
	homePrefix := home
	if !strings.HasSuffix(homePrefix, "/") {
		homePrefix += "/"
	}
	if path == home || strings.HasPrefix(path, homePrefix) {
		return
	}
	for _, p := range ephemeralPathPrefixes {
		if strings.HasPrefix(path, p) {
			return // ephemeral system dir, never bind-mount
		}
	}

	pathMountAttemptsMu.Lock()
	if last, ok := pathMountAttempts[path]; ok && time.Since(last.when) < last.debounce {
		pathMountAttemptsMu.Unlock()
		return // already attempted recently; refuse to cascade restart again
	}
	// Pre-stamp with the success debounce so concurrent calls during the
	// in-flight reload+restart see a long window and skip — prevents the
	// cascade. If anything fails below, we overwrite with the short
	// failure debounce so the next caller retries instead of waiting 60s.
	pathMountAttempts[path] = pathMountStamp{when: time.Now(), debounce: pathMountSuccessDebounce}
	pathMountAttemptsMu.Unlock()

	versions, _ := listInstalledPHPVersions()

	// Collect all quadlet files to check: FPM containers + nginx.
	type quadletInfo struct {
		unitName string
		path     string
	}
	var quadlets []quadletInfo
	for _, v := range versions {
		short := strings.ReplaceAll(v, ".", "")
		unitName := "lerd-php" + short + "-fpm"
		quadlets = append(quadlets, quadletInfo{unitName, filepath.Join(config.QuadletDir(), unitName+".container")})
	}
	quadlets = append(quadlets, quadletInfo{"lerd-nginx", filepath.Join(config.QuadletDir(), "lerd-nginx.container")})

	var changedUnits []string
	for _, q := range quadlets {
		existing, readErr := os.ReadFile(q.path)
		if readErr != nil {
			continue
		}

		volumePrefix := fmt.Sprintf("Volume=%s:%s:", path, path)
		if strings.Contains(string(existing), volumePrefix) {
			continue
		}

		updated := InjectExtraVolumes(string(existing), []string{path})
		if updated == string(existing) {
			continue
		}
		// Route through WriteQuadletDiff (not os.WriteFile) so the same
		// transformations every other writer applies — BindForLAN,
		// PairIPv6Binds, StripInstallSection, PlatformPodmanArgs — and the
		// platform sync hook (AfterQuadletWriteFn, which keeps the macOS
		// launchd plist consistent with the .container file) run here too.
		// Without this, lazy mount injection on macOS used to update the
		// quadlet without ever syncing the plist that launchd actually runs.
		changed, writeErr := WriteQuadletDiff(q.unitName, updated)
		if writeErr != nil {
			continue
		}
		if changed {
			changedUnits = append(changedUnits, q.unitName)
		}
	}

	if len(changedUnits) > 0 {
		var failed bool
		if err := DaemonReload(); err != nil {
			fmt.Fprintf(os.Stderr, "lerd: daemon-reload after mounting %s failed: %v\n", path, err)
			failed = true
		}
		for _, unit := range changedUnits {
			if err := RestartUnit(unit); err != nil {
				fmt.Fprintf(os.Stderr, "lerd: restart %s after mounting %s failed: %v\n", unit, path, err)
				failed = true
				continue
			}
			// Belt and suspenders: even when reload+restart both report
			// success, the running container can end up without the new
			// mount (observed in production — see oracle.X release notes
			// for the runc chdir-failed incident on /home/unimedvr/...).
			// Verify the destination is actually present; if not, mark
			// failed so the debounce shortens and a retry happens fast.
			if ContainerRunningQuiet(unit) {
				if has, err := ContainerHasMount(unit, path); err == nil && !has {
					fmt.Fprintf(os.Stderr, "lerd: %s reload+restart reported success but mount %s is still missing — retry will fire shortly\n", unit, path)
					failed = true
				}
			}
		}
		if failed {
			// Shorten the debounce so the next caller retries instead of
			// silently early-returning for 60s with a stale-mount container.
			pathMountAttemptsMu.Lock()
			pathMountAttempts[path] = pathMountStamp{when: time.Now(), debounce: pathMountFailureDebounce}
			pathMountAttemptsMu.Unlock()
		}
	}
}

// EnsureUserIni creates the per-version user php.ini with defaults if it doesn't exist.
// Same bind-mount race as EnsureXdebugIni: when this path is missing at FPM
// container start time, podman auto-creates it as a directory and the next
// EnsureUserIni call (which only Stat'd, didn't IsDir-check) silently no-ops
// while the user's php.ini is never written. Heal stale directories before
// returning the no-op fast path.
func EnsureUserIni(version string) error {
	path := config.PHPUserIniFile(version)
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return nil // already a regular file
		}
		if rmErr := os.Remove(path); rmErr != nil {
			return fmt.Errorf("removing stale user ini directory: %w", rmErr)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	content := "; Lerd per-version PHP settings for PHP " + version + "\n" +
		"; Edit this file, then restart: systemctl --user restart lerd-php" +
		strings.ReplaceAll(version, ".", "") + "-fpm\n" +
		";\n" +
		"; memory_limit = 512M\n" +
		"; upload_max_filesize = 64M\n" +
		"; post_max_size = 64M\n" +
		"; max_execution_time = 60\n"
	return os.WriteFile(path, []byte(content), 0644)
}
