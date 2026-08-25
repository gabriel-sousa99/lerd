package siteops

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// RemoveWorktreeVhosts removes every worktree subdomain vhost belonging to the
// given primary domain and returns the domains it removed. A conf whose domain
// is a registered site of its own (a group secondary at <label>.<primary>) is
// left alone: it matches the suffix scan but is not a worktree.
func RemoveWorktreeVhosts(primaryDomain string) []string {
	confD := config.NginxConfD()
	entries, err := os.ReadDir(confD)
	if err != nil {
		return nil
	}
	suffix := "." + primaryDomain + ".conf"
	var removed []string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		domain := strings.TrimSuffix(e.Name(), ".conf")
		if _, err := config.FindSiteByDomain(domain); err == nil {
			continue
		}
		_ = os.Remove(filepath.Join(confD, e.Name()))
		removed = append(removed, domain)
	}
	return removed
}
