// Package dashboard resolves the address the lerd web UI is opened at.
package dashboard

import (
	"net/http"
	"time"
)

const (
	// VhostURL is where the dashboard normally lives: the nginx-proxied vhost,
	// so it shows up under a name rather than a bare IP:port.
	VhostURL = "http://lerd.localhost"
	// DirectURL is lerd-ui's own port. nginx is a container `lerd stop` stops,
	// so this is the only address that still answers on a stopped stack.
	DirectURL = "http://127.0.0.1:7073"
)

// probe is the reachability check, swapped out in tests.
var probe = serving

// Serving reports whether the stack is up, by asking the vhost for the
// dashboard page. nginx is a container `lerd stop` stops, so a vhost that does
// not answer means lerd is down.
func Serving() bool { return probe(VhostURL) }

// URL returns the address to open. It prefers the vhost and falls back to
// lerd-ui's own port when nginx is not serving, which is what lets "open the
// dashboard" land on a working page while lerd is stopped.
func URL() string {
	if Serving() {
		return VhostURL
	}
	return DirectURL
}

// serving reports whether base hands out the dashboard page. It asks for the
// page itself rather than an API route because the vhost proxies the API to
// lerd-ui only on some platforms, while the page is what every install serves.
// The timeout is short: this sits in front of a window opening.
func serving(base string) bool {
	c := &http.Client{Timeout: 900 * time.Millisecond}
	resp, err := c.Get(base + "/")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < http.StatusInternalServerError
}
