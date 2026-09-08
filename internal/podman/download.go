package podman

import "github.com/gabriel-sousa99/lerd/internal/imagepull"

// PendingDownload is what an operation would fetch before it can run, so a UI
// can disclose the cost and let the user decline it. A zero value means the
// operation downloads nothing.
type PendingDownload struct {
	Image string `json:"image"`
	Bytes int64  `json:"bytes"`
	Local bool   `json:"local"`
}

// DescribeDownload sizes an image reference from the registry unless it is
// already in the local store, in which case there is nothing to disclose.
func DescribeDownload(image string) PendingDownload {
	if image == "" {
		return PendingDownload{}
	}
	if ImageExists(image) {
		return PendingDownload{Image: image, Local: true}
	}
	bytes, _ := imagepull.Size(image)
	return PendingDownload{Image: image, Bytes: bytes}
}

// PHPDownload reports what building or rebuilding the PHP image for version
// would fetch: the prebuilt base the build layers on top of. A derived
// FrankenPHP image for the same version pulls its own base on top of this,
// which only matters for a version that actually runs an Octane site.
func PHPDownload(version string) PendingDownload {
	return DescribeDownload(PHPBaseImageRef(version))
}
