// Package origin centralises every URL lerd fetches its own assets from: release
// binaries, the framework and service stores, the changelog, and the GHCR base
// images. Each endpoint is overridable via its environment variable for tests
// and mirrors.
//
// Oracle fork: the artefacts this repository publishes itself — releases, the
// changelog and the PHP-FPM base images, which carry the Oracle Instant Client
// and oci8 — are served from the fork. The framework and service stores are not
// forked, so they keep pointing at upstream's lerd-env org and stay current.
package origin

import (
	"os"
	"strings"
)

const (
	owner          = "gabriel-sousa99"      // GHCR namespace for the Oracle base images
	mainRepo       = "gabriel-sousa99/lerd" // releases, installer, changelog
	frameworksRepo = "lerd-env/frameworks"
	servicesRepo   = "lerd-env/services"

	// defaultBranch is the fork's default branch, where the Oracle work lives.
	// The fork's `main` still tracks upstream, so anything fetched by raw URL
	// has to name this branch explicitly or it silently serves upstream's file.
	defaultBranch = "oracle-oci8-support"
)

// StoreBaseURLs returns the framework-store base. The definitions live under a
// frameworks/ subdir (index.json + <name>.yaml), not at the repo root.
func StoreBaseURLs() []string {
	if list := splitList(os.Getenv("LERD_STORE_BASE_URL")); len(list) > 0 {
		return list
	}
	return []string{"https://raw.githubusercontent.com/" + frameworksRepo + "/main/frameworks"}
}

// ServiceStoreBaseURLs returns the service-preset-store base, nested under a
// services/ subdir.
func ServiceStoreBaseURLs() []string {
	if list := splitList(os.Getenv("LERD_SERVICES_BASE_URL")); len(list) > 0 {
		return list
	}
	return []string{"https://raw.githubusercontent.com/" + servicesRepo + "/main/services"}
}

// ReleaseBaseURLs lists GitHub releases bases.
func ReleaseBaseURLs() []string {
	if list := splitList(os.Getenv("LERD_RELEASES_URL")); len(list) > 0 {
		return list
	}
	return []string{"https://github.com/" + mainRepo + "/releases"}
}

// ReleaseDownloadBases lists release-asset download bases.
func ReleaseDownloadBases() []string {
	if list := splitList(os.Getenv("LERD_RELEASE_DOWNLOAD_URL")); len(list) > 0 {
		return list
	}
	out := ReleaseBaseURLs()
	for i := range out {
		out[i] += "/download"
	}
	return out
}

// ReleaseAPIBaseURLs lists GitHub API bases.
func ReleaseAPIBaseURLs() []string {
	if list := splitList(os.Getenv("LERD_RELEASES_API_URL")); len(list) > 0 {
		return list
	}
	return []string{"https://api.github.com/repos/" + mainRepo}
}

// ChangelogURLs lists raw changelog URLs.
//
// Oracle fork: two corrections over upstream's path. The fork's default branch
// is oracle-oci8-support, not main (main tracks upstream and carries none of the
// fork's entries), and the repo-root CHANGELOG.md is a symlink — raw.github
// serves a symlink's target *path* as the body, so that URL returned the literal
// string "docs/changelog.md" with a 200 and `lerd whatsnew` rendered nothing.
// Fetch the real file on the real branch instead.
func ChangelogURLs() []string {
	if list := splitList(os.Getenv("LERD_CHANGELOG_URL")); len(list) > 0 {
		return list
	}
	return []string{"https://raw.githubusercontent.com/" + mainRepo + "/" + defaultBranch + "/docs/changelog.md"}
}

// BaseImageRefs lists GHCR refs for a prebuilt PHP-FPM base image, where phpShort
// is the dotless version (e.g. "85") and hash pins the image to the embedded
// Containerfile template.
func BaseImageRefs(phpShort, hash string) []string {
	suffix := "/lerd-php" + phpShort + "-fpm-base:" + hash
	if v := os.Getenv("LERD_BASE_IMAGE_REGISTRY"); v != "" {
		return []string{v + suffix}
	}
	return []string{"ghcr.io/" + owner + suffix}
}

// splitList parses a comma-separated override into trimmed, non-empty entries.
func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
