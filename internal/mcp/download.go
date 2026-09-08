package mcp

import (
	"fmt"

	"github.com/gabriel-sousa99/lerd/internal/imagepull"
	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// An action that has to fetch a container image says so before a single byte
// moves, the way the CLI and the dashboard do. The assistant calling the tool
// is not the one paying for the bandwidth, so the size is reported back for it
// to relay and the pull only starts on a second call carrying confirm: true.

// downloadGate returns the disclosure to answer with when the action would
// pull, and nil when it would not: an image already in the local store, an
// operation with nothing to fetch, or an estimate the registry did not answer
// for, none of which are worth stopping for. what names the operation the way
// the caller asked for it.
func downloadGate(args map[string]any, pending podman.PendingDownload, what string) map[string]any {
	if pending.Image == "" || pending.Local || boolArg(args, "confirm") {
		return nil
	}
	size := imagepull.Note(pending.Bytes)
	if size == "" {
		size = " (size unknown, the registry did not answer)"
	}
	return toolErr(fmt.Sprintf("%s downloads the container image %s%s, which is not on this machine yet. Nothing has been downloaded. Tell the user what it costs, then re-run this call with confirm: true.", what, pending.Image, size))
}
