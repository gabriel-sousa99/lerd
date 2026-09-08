package mcp

import (
	"github.com/gabriel-sousa99/lerd/internal/dns"
)

func execDNSDiagnose(args map[string]any) (any, *rpcError) {
	tld := strArg(args, "tld")
	if tld == "" {
		tld = dns.ConfiguredTLD()
	}
	return toolJSON(dns.Diagnose(tld)), nil
}
