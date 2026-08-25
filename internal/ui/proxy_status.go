package ui

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/podman"
	"github.com/gabriel-sousa99/lerd/internal/proxyops"
	"github.com/gabriel-sousa99/lerd/internal/reqstats"
)

const proxyProbeTimeout = 2 * time.Second

type proxyStatusResponse struct {
	State              string  `json:"state"`
	UpstreamReachable  bool    `json:"upstream_reachable"`
	LatencyMillis      float64 `json:"latency_ms"`
	HTTPStatus         int     `json:"http_status,omitempty"`
	CheckedAt          string  `json:"checked_at"`
	NginxRunning       bool    `json:"nginx_running"`
	VhostPresent       bool    `json:"vhost_present"`
	CertificatePresent bool    `json:"certificate_present"`
	UnitState          string  `json:"unit_state,omitempty"`
	Error              string  `json:"error,omitempty"`
}

type proxyConfigResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

var proxyNginxRunning = func() bool {
	running, err := podman.ContainerRunning("lerd-nginx")
	return err == nil && running
}

var proxyManagedUnitStatus = podman.UnitStatus

func handleProxyStatus(w http.ResponseWriter, p config.Proxy) {
	status := proxyRuntimeStatus(p)
	writeProxyJSON(w, http.StatusOK, status)
}

func proxyRuntimeStatus(p config.Proxy) proxyStatusResponse {
	vhostPath := proxyVhostPath(p)
	_, vhostErr := os.Stat(vhostPath)
	_, certErr := os.Stat(filepath.Join(config.CertsDir(), "sites", p.PrimaryDomain()+".crt"))

	status := proxyStatusResponse{
		State:              "healthy",
		CheckedAt:          time.Now().UTC().Format(time.RFC3339),
		NginxRunning:       proxyNginxRunning(),
		VhostPresent:       vhostErr == nil,
		CertificatePresent: p.Secured && certErr == nil,
	}

	if p.Managed {
		status.UnitState, _ = proxyManagedUnitStatus(proxyops.ManagedUnitName(p.Name))
	}
	if p.Paused {
		status.State = "paused"
		return status
	}
	if p.Managed && status.UnitState != "active" {
		if status.UnitState == "failed" {
			status.State = "failed"
		} else {
			status.State = "inactive"
		}
		return status
	}
	if !status.NginxRunning || !status.VhostPresent || (p.Secured && !status.CertificatePresent) {
		status.State = "misconfigured"
	}

	reachable, latency, httpStatus, err := probeProxyUpstream(p)
	status.UpstreamReachable = reachable
	status.LatencyMillis = float64(latency.Microseconds()) / 1000
	status.HTTPStatus = httpStatus
	if err != nil {
		status.Error = err.Error()
		if status.State == "healthy" {
			status.State = "unreachable"
		}
	} else if httpStatus >= http.StatusBadRequest && status.State == "healthy" {
		status.State = "degraded"
	}
	return status
}

func probeProxyUpstream(p config.Proxy) (bool, time.Duration, int, error) {
	if p.Site != "" && p.UpstreamPort == 0 {
		scheme := "http"
		if p.Secured {
			scheme = "https"
		}
		path := p.HealthPath
		if path == "" {
			path = "/"
		}
		return probeProxyHTTP(fmt.Sprintf("%s://%s%s", scheme, p.PrimaryDomain(), path), scheme)
	}

	host := p.UpstreamHost
	if host == "" || host == "host.containers.internal" {
		host = "127.0.0.1"
	}
	address := net.JoinHostPort(host, fmt.Sprintf("%d", p.UpstreamPort))
	started := time.Now()

	if p.HealthPath == "" {
		conn, err := net.DialTimeout("tcp", address, proxyProbeTimeout)
		latency := time.Since(started)
		if err != nil {
			return false, latency, 0, err
		}
		_ = conn.Close()
		return true, latency, 0, nil
	}

	return probeProxyHTTP(fmt.Sprintf("%s://%s%s", p.EffectiveUpstreamScheme(), address, p.HealthPath), p.EffectiveUpstreamScheme())
}

func probeProxyHTTP(url, scheme string) (bool, time.Duration, int, error) {
	started := time.Now()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if scheme == "https" {
		// Nginx does not verify upstream certificates by default; the probe
		// mirrors that behaviour for local self-signed development servers.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	client := &http.Client{Timeout: proxyProbeTimeout, Transport: transport}
	resp, err := client.Get(url)
	latency := time.Since(started)
	if err != nil {
		return false, latency, 0, err
	}
	defer resp.Body.Close()
	return true, latency, resp.StatusCode, nil
}

func handleProxyGeneratedConfig(w http.ResponseWriter, p config.Proxy) {
	path := proxyVhostPath(p)
	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeProxyJSON(w, http.StatusOK, proxyConfigResponse{Path: path, Content: string(content)})
}

func handleProxyStats(w http.ResponseWriter, p config.Proxy) {
	key := reqstats.ProxyKey(p.Name)
	stats, ok := reqstats.LoadSite(config.RequestStatsFile(), key)
	if !ok {
		stats = reqstats.SiteStats{Site: key, Slow: []reqstats.RouteStat{}}
	}
	writeProxyJSON(w, http.StatusOK, stats)
}

func proxyVhostPath(p config.Proxy) string {
	suffix := ".conf"
	if p.Secured {
		suffix = "-ssl.conf"
	}
	return filepath.Join(config.NginxConfD(), p.PrimaryDomain()+suffix)
}
