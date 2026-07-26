package mcp

import (
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/dns"
)

func execDNSDiagnose(args map[string]any) (any, *rpcError) {
	tld := strArg(args, "tld")
	if tld == "" {
		if cfg, _ := config.LoadGlobal(); cfg != nil {
			tld = cfg.DNS.TLD
		}
	}
	return toolJSON(dns.Diagnose(tld)), nil
}
