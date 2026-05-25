package proxyops

import (
	"regexp"
	"strings"
)

var gTLDs = []string{
	".com", ".net", ".org", ".info", ".biz", ".dev", ".app",
	".tech", ".site", ".online", ".store", ".shop", ".xyz",
	".cloud", ".digital", ".studio", ".agency", ".host", ".ltd",
	".localhost", ".test",
}

var ccTLDPattern = regexp.MustCompile(`\.[a-z]{2}$`)

// ProxyNameAndDomain derives a clean proxy name and FQDN from a raw input
// (directory name or user-supplied label) and a default TLD. Trailing TLD
// in the input is stripped; remaining dots become dashes.
func ProxyNameAndDomain(raw, tld string) (string, string) {
	name := strings.ToLower(raw)
	if stripped, ok := stripGTLD(name); ok {
		name = stripped
	} else if m := ccTLDPattern.FindStringIndex(name); m != nil {
		name = name[:m[0]]
	}
	name = strings.ReplaceAll(name, ".", "-")
	if name == "" {
		name = "proxy"
	}
	return name, name + "." + tld
}

func stripGTLD(name string) (string, bool) {
	for _, ext := range gTLDs {
		if strings.HasSuffix(name, ext) {
			return name[:len(name)-len(ext)], true
		}
	}
	return name, false
}
