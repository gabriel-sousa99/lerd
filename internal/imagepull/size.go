package imagepull

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// acceptManifests lists every manifest media type a registry may answer with.
// Without all four, Docker Hub returns a v1 manifest with no layer sizes.
const acceptManifests = "application/vnd.oci.image.index.v1+json," +
	"application/vnd.docker.distribution.manifest.list.v2+json," +
	"application/vnd.oci.image.manifest.v1+json," +
	"application/vnd.docker.distribution.manifest.v2+json"

var httpClient = &http.Client{Timeout: 6 * time.Second}

// sizeCache memoises lookups for the life of the process: a single `lerd start`
// asks for the same base image once per PHP version.
var sizeCache sync.Map // ref -> int64, 0 meaning "unknown"

// Size returns the compressed download size of ref in bytes, read from the
// registry manifest without pulling anything. ok is false for a locally built
// image and whenever the registry cannot be reached or understood, in which
// case callers disclose the pull without a number rather than failing.
func Size(ref string) (int64, bool) {
	if cached, hit := sizeCache.Load(ref); hit {
		n := cached.(int64)
		return n, n > 0
	}
	n := lookupSize(ref)
	sizeCache.Store(ref, n)
	return n, n > 0
}

func lookupSize(ref string) int64 {
	host, repo, tag, ok := parseRef(ref)
	if !ok {
		return 0
	}
	n, _ := sizeFrom("https://"+host, repo, tag, 0)
	return n
}

// sizeFrom sums the config blob and every layer of the manifest for repo@tag,
// following a multi-arch index one level down to this host's architecture.
func sizeFrom(base, repo, tag string, depth int) (int64, bool) {
	if depth > 1 {
		return 0, false
	}
	body, err := fetchManifest(base, repo, tag)
	if err != nil {
		return 0, false
	}
	var m struct {
		MediaType string `json:"mediaType"`
		Manifests []struct {
			Digest   string `json:"digest"`
			Platform struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			} `json:"platform"`
		} `json:"manifests"`
		Config struct {
			Size int64 `json:"size"`
		} `json:"config"`
		Layers []struct {
			Size int64 `json:"size"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return 0, false
	}
	if len(m.Manifests) > 0 {
		digest := ""
		for _, sub := range m.Manifests {
			if sub.Platform.OS == "linux" && sub.Platform.Architecture == runtime.GOARCH {
				digest = sub.Digest
				break
			}
		}
		if digest == "" {
			return 0, false
		}
		return sizeFrom(base, repo, digest, depth+1)
	}
	total := m.Config.Size
	for _, l := range m.Layers {
		total += l.Size
	}
	if total == 0 {
		return 0, false
	}
	return total, true
}

// fetchManifest GETs a manifest, doing the anonymous bearer-token dance that
// Docker Hub and ghcr.io both require for public images.
func fetchManifest(base, repo, tag string) ([]byte, error) {
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", base, repo, tag)
	body, challenge, err := get(url, "")
	if err != nil {
		return nil, err
	}
	if challenge == "" {
		return body, nil
	}
	token, err := fetchToken(challenge, repo)
	if err != nil {
		return nil, err
	}
	body, challenge, err = get(url, token)
	if err != nil {
		return nil, err
	}
	if challenge != "" {
		return nil, fmt.Errorf("registry still refuses anonymous access to %s", repo)
	}
	return body, nil
}

// get returns the response body, or the WWW-Authenticate header when the
// registry answered 401 and wants a token first.
func get(url, token string) (body []byte, challenge string, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", acceptManifests)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		if c := resp.Header.Get("Www-Authenticate"); strings.HasPrefix(c, "Bearer ") {
			return nil, c, nil
		}
		return nil, "", fmt.Errorf("unauthorized: %s", url)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("registry returned %s for %s", resp.Status, url)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return b, "", err
}

var challengeField = regexp.MustCompile(`([a-zA-Z]+)="([^"]*)"`)

func fetchToken(challenge, repo string) (string, error) {
	fields := map[string]string{}
	for _, m := range challengeField.FindAllStringSubmatch(challenge, -1) {
		fields[strings.ToLower(m[1])] = m[2]
	}
	realm := fields["realm"]
	if realm == "" {
		return "", fmt.Errorf("auth challenge without a realm")
	}
	scope := fields["scope"]
	if scope == "" {
		scope = "repository:" + repo + ":pull"
	}
	req, err := http.NewRequest(http.MethodGet, realm, nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	if s := fields["service"]; s != "" {
		q.Set("service", s)
	}
	q.Set("scope", scope)
	req.URL.RawQuery = q.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %s", resp.Status)
	}
	var t struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&t); err != nil {
		return "", err
	}
	if t.Token != "" {
		return t.Token, nil
	}
	if t.AccessToken != "" {
		return t.AccessToken, nil
	}
	return "", fmt.Errorf("token endpoint returned no token")
}

// parseRef splits an image reference into the registry host, the full
// repository path and the tag or digest to ask for. ok is false for anything
// that lives only in the local store, which has no registry to ask.
func parseRef(ref string) (host, repo, tag string, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", "", false
	}
	host = "registry-1.docker.io"
	rest := ref
	if i := strings.Index(ref, "/"); i > 0 {
		first := ref[:i]
		if first == "localhost" || strings.ContainsAny(first, ".:") {
			host = first
			rest = ref[i+1:]
		}
	}
	if host == "localhost" {
		return "", "", "", false
	}
	if host == "docker.io" || host == "index.docker.io" {
		host = "registry-1.docker.io"
	}

	tag = "latest"
	if at := strings.Index(rest, "@"); at >= 0 {
		tag, rest = rest[at+1:], rest[:at]
	} else if colon := strings.LastIndex(rest, ":"); colon >= 0 && !strings.Contains(rest[colon:], "/") {
		tag, rest = rest[colon+1:], rest[:colon]
	}
	// ":local" is lerd's own tag for images it builds on the machine.
	if tag == "local" {
		return "", "", "", false
	}
	if host == "registry-1.docker.io" && !strings.Contains(rest, "/") {
		rest = "library/" + rest
	}
	return host, rest, tag, rest != ""
}
