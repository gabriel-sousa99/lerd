package ui

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/proxyops"
)

// proxyDTO is the JSON shape returned by /api/proxies* endpoints. It
// flattens config.Proxy plus a derived `domain` (PrimaryDomain) so the
// frontend can render the proxy list without extra roundtrips.
type proxyDTO struct {
	Name         string   `json:"name"`
	Domain       string   `json:"domain"`
	Domains      []string `json:"domains"`
	UpstreamPort int      `json:"upstream_port"`
	UpstreamHost string   `json:"upstream_host"`
	Path         string   `json:"path,omitempty"`
	Secured      bool     `json:"secured"`
	Paused       bool     `json:"paused"`
	Managed      bool     `json:"managed"`
	NodeVersion  string   `json:"node_version,omitempty"`
	Command      string   `json:"cmd,omitempty"`
	AutoStart    bool     `json:"autostart"`
}

func toProxyDTO(p config.Proxy) proxyDTO {
	host := p.UpstreamHost
	if host == "" {
		host = "host.containers.internal"
	}
	return proxyDTO{
		Name:         p.Name,
		Domain:       p.PrimaryDomain(),
		Domains:      p.Domains,
		UpstreamPort: p.UpstreamPort,
		UpstreamHost: host,
		Path:         p.Path,
		Secured:      p.Secured,
		Paused:       p.Paused,
		Managed:      p.Managed,
		NodeVersion:  p.NodeVersion,
		Command:      p.Command,
		AutoStart:    p.AutoStart,
	}
}

// handleProxies serves GET /api/proxies (list) and POST /api/proxies (create).
func handleProxies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		reg, err := config.LoadProxies()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]proxyDTO, 0, len(reg.Proxies))
		for _, p := range reg.Proxies {
			out = append(out, toProxyDTO(p))
		}
		writeProxyJSON(w, http.StatusOK, out)
	case http.MethodPost:
		var body struct {
			Domain      string `json:"domain"`
			Port        int    `json:"port"`
			Path        string `json:"path"`
			NoSecure    bool   `json:"no_secure"`
			Managed     bool   `json:"managed"`
			Command     string `json:"cmd"`
			NodeVersion string `json:"node_version"`
			AutoStart   bool   `json:"autostart"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p, err := proxyops.Add(proxyops.AddOptions{
			Domain:      body.Domain,
			Port:        body.Port,
			Path:        body.Path,
			NoSecure:    body.NoSecure,
			Managed:     body.Managed,
			Command:     body.Command,
			NodeVersion: body.NodeVersion,
			AutoStart:   body.AutoStart,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeProxyJSON(w, http.StatusCreated, toProxyDTO(p))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProxyAction serves /api/proxies/{name}[/action] where action is one
// of secure, unsecure, pause, resume, start, stop. DELETE on the bare name
// removes the proxy. All actions return the updated proxyDTO except delete
// which returns 204.
func handleProxyAction(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/proxies/")
	parts := strings.SplitN(rest, "/", 2)
	if parts[0] == "" {
		http.Error(w, "missing proxy name", http.StatusBadRequest)
		return
	}
	name := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if r.Method == http.MethodDelete && action == "" {
		if err := proxyops.Remove(name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p, err := config.FindProxy(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	switch action {
	case "secure":
		if err := proxyops.SetSecured(p, true); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case "unsecure":
		if err := proxyops.SetSecured(p, false); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case "pause":
		p.Paused = true
		if err := proxyops.ApplyPause(p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case "resume":
		p.Paused = false
		if err := proxyops.ApplyPause(p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case "start":
		if err := proxyops.WriteManagedQuadlet(*p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := proxyops.StartManaged(p.Name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case "stop":
		if err := proxyops.StopManaged(p.Name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "ação desconhecida", http.StatusNotFound)
		return
	}
	writeProxyJSON(w, http.StatusOK, toProxyDTO(*p))
}

// writeProxyJSON is a small helper because the package-level writeJSON
// doesn't accept a status code; proxy endpoints need to emit 201/200/204.
func writeProxyJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
