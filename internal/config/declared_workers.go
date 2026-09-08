package config

// DeclaredWorkerNames returns every worker name a site still answers for: the
// ones its framework definition names, the custom workers in its .lerd.yaml,
// and the lerd-managed builtins (the Stripe listener, the host-proxy dev
// server). A gated worker counts as declared, because a check that stops
// matching only hides the worker from the list while the definition still
// names it.
//
// ok is false when nothing could be resolved to compare against, so a caller
// that cannot tell declared from undeclared leaves every unit alone rather than
// calling all of them stale.
func DeclaredWorkerNames(s Site, fw *Framework) (names map[string]bool, ok bool) {
	proj, _ := LoadProjectConfig(s.Path)
	if fw == nil && (proj == nil || len(proj.CustomWorkers) == 0) {
		return nil, false
	}
	names = map[string]bool{}
	if fw != nil {
		for n := range fw.Workers {
			names[n] = true
		}
	}
	if proj != nil {
		for n := range proj.CustomWorkers {
			names[n] = true
		}
	}
	// The builtins are declared by lerd itself rather than by any definition, so
	// they belong in the set that every caller compares against.
	names[StripeWorkerName] = true
	names[HostProxyWorkerName] = true
	return names, true
}
