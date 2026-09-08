package ui

import (
	"os"
	"strings"
	"testing"
)

// Both PHP build handlers leave the same things stale: the container states,
// the patch probed out of the image that was just replaced, and the status
// snapshot built from both. Install used to poll only the container cache, so
// a freshly built version reported no patch until an unrelated background
// probe happened to fill it in. Guarding the source keeps the two paths
// together, since neither handler can be driven from a test without building
// a real image.
func TestPHPBuildHandlersRefreshThroughOneHelper(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("reading server.go: %v", err)
	}
	for _, handler := range []string{"func handlePHPInstall(", "func handlePHPRebuild("} {
		body := funcBody(string(src), handler)
		if body == "" {
			t.Fatalf("%s not found in server.go", handler)
		}
		if !strings.Contains(body, "refreshAfterPHPBuild(version)") {
			t.Errorf("%s does not refresh through refreshAfterPHPBuild", handler)
		}
		if strings.Contains(body, "podman.Cache.PollNow()") {
			t.Errorf("%s polls containers on its own instead of using refreshAfterPHPBuild", handler)
		}
	}
}

// funcBody returns the source between a function's opening line and the next
// top-level closing brace.
func funcBody(src, decl string) string {
	i := strings.Index(src, decl)
	if i < 0 {
		return ""
	}
	rest := src[i:]
	if end := strings.Index(rest, "\n}\n"); end >= 0 {
		return rest[:end]
	}
	return rest
}
