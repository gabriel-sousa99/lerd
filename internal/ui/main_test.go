package ui

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/push"

	"github.com/gabriel-sousa99/lerd/internal/desktopnotify"
)

// TestMain neutralises both notification sinks for the whole package so no test
// can fire one at the developer running the suite, even through the async
// goroutine in dispatchNotification that can outlive a single test. The browser
// half was already covered; the desktop half reached a real notification daemon,
// so a run of the site flows raised "Create finished" and "Link failed" popups
// naming the test's own temp directories.
func TestMain(m *testing.M) {
	push.HTTPClient = &http.Client{Transport: discardPushTransport{}}
	emitDesktopNotification = func(desktopnotify.Request) (uint32, error) { return 0, nil }
	os.Exit(m.Run())
}

type discardPushTransport struct{}

func (discardPushTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}, nil
}
