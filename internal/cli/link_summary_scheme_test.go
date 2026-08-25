package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The link registers a `secured: true` project as secured and issues its
// certificate before it reports anything, so the summary has to name the scheme
// the site is actually served on (#1431).
func TestPrintLinkSummary_reportsTheSchemeTheSiteIsServedOn(t *testing.T) {
	for _, c := range []struct {
		name    string
		secured bool
		want    string
	}{
		{"a secured site is reported on https", true, "https://app.test"},
		{"a plain site is reported on http", false, "http://app.test"},
	} {
		t.Run(c.name, func(t *testing.T) {
			defer func(prev bool) { linkApplied = prev }(linkApplied)
			site := config.Site{Name: "app", Domains: []string{"app.test"}, Path: t.TempDir(), Secured: c.secured}
			out, err := runCapturingStdout(func() error {
				printLinkSummary(site, time.Now(), false)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), c.want) {
				t.Errorf("summary does not mention %s:\n%s", c.want, out)
			}
		})
	}
}
