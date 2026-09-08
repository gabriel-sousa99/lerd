package config

import "strings"

// ValidPHPVersion reports whether s is a PHP version in MAJOR.MINOR form,
// rejecting "8,5" and plain words. The init prompt, the MCP argument check and
// the project-config doctor all ask the same question, so they ask it here.
func ValidPHPVersion(s string) bool {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return false
	}
	for _, p := range parts {
		if p == "" || strings.TrimLeft(p, "0123456789") != "" {
			return false
		}
	}
	return true
}
