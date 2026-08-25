package cli

import "github.com/gabriel-sousa99/lerd/internal/siteops"

// Wire the dev server refresh to the siteops paths that move a site's addresses,
// securing it and changing its domains, which the CLI, the UI and MCP all share.
// The mechanism lives in this package, so the hook is filled in from here.
func init() {
	siteops.RefreshDevServers = RefreshDevServers
	siteops.StopSiteShares = stopSiteShares
}

// stopSiteShares releases every listener a site holds, its worktrees' included,
// so unlinking it cannot leave one bound with no registry entry left to reach it
// by. Worktree shares are keyed per branch, so they need releasing by name.
func stopSiteShares(siteName string) {
	LANShareStopServer(siteName)
	LANShareStopWorktrees(siteName)
	PublicShareStopServer(siteName)
	StopSiteTunnels(siteName)
}
