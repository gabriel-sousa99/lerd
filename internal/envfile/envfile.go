// Package envfile provides helpers for reading and updating .env files
// while preserving comments, blank lines, and line order.
package envfile

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ApplyUpdates rewrites the .env at path, replacing values for any key in updates.
// Keys not already present are appended at the end in stable (sorted) order so
// idempotent calls produce identical bytes regardless of Go's map-range
// nondeterminism. Comments and blank lines are preserved. The write is skipped
// when the resulting contents match the existing file, so dev-side watchers
// (vite, IDE indexers, opcache) don't see mtime churn on idempotent calls.
// The file's existing mode is preserved.
//
// Keys must not contain '=' or any newline character; values must not contain
// newline characters. These checks reject the env_overrides injection vector
// where a malicious .lerd.yaml value containing "\nADMIN_TOKEN=stolen" would
// otherwise split a single .env line into two and silently introduce an
// unrelated key.
func ApplyUpdates(path string, updates map[string]string) error {
	for k, v := range updates {
		if err := validateEnvKey(k); err != nil {
			return err
		}
		if err := validateEnvValue(v); err != nil {
			return fmt.Errorf("value for %q: %w", k, err)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// This writer appends `key=value` lines, which is nonsense in a PHP settings
	// file: the file stops parsing and every request to the site fails. A caller
	// that knows the format goes through ApplyUpdatesIn; one that resolved a path
	// without carrying the format is stopped here rather than breaking the site.
	if isPHPSource(original) {
		return fmt.Errorf("%s holds PHP, not dotenv lines; write it through ApplyUpdatesIn with the framework's format", filepath.Base(path))
	}

	var lines []string
	applied := map[string]bool{}

	scanner := bufio.NewScanner(strings.NewReader(string(original)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") && strings.Contains(line, "=") {
			k, _, _ := strings.Cut(line, "=")
			k = strings.TrimSpace(k)
			if newVal, ok := updates[k]; ok {
				line = k + "=" + newVal
				applied[k] = true
			}
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Stable iteration order: a Go map's range is randomised, so the first
	// write of N new keys could produce a different byte ordering each
	// run. Sorted append makes the output reproducible and lets the
	// "skip if unchanged" mtime guard at the end of this function actually
	// do its job on the second call.
	pending := make([]string, 0, len(updates))
	for k := range updates {
		if applied[k] {
			continue
		}
		pending = append(pending, k)
	}
	sort.Strings(pending)
	for _, k := range pending {
		v := updates[k]
		// Look for a commented-out version to uncomment in place.
		found := false
		for i, line := range lines {
			if !strings.HasPrefix(line, "#") {
				continue
			}
			trimmed := strings.TrimLeft(line, "# ")
			if !strings.Contains(trimmed, "=") {
				continue
			}
			ck, _, _ := strings.Cut(trimmed, "=")
			if strings.TrimSpace(ck) == k {
				lines[i] = k + "=" + v
				found = true
				break
			}
		}
		if !found {
			lines = append(lines, k+"="+v)
		}
	}

	out := strings.Join(lines, "\n")
	if len(lines) > 0 && len(original) > 0 && original[len(original)-1] == '\n' {
		out += "\n"
	} else if len(lines) > 0 && len(original) == 0 {
		out += "\n"
	}
	if out == string(original) {
		return nil
	}
	return writeFile(path, []byte(out), info.Mode().Perm())
}

// validateEnvKey rejects keys that would corrupt .env structure if written
// verbatim: empty keys, keys containing '=', or keys with embedded newlines.
func validateEnvKey(k string) error {
	if k == "" {
		return fmt.Errorf("invalid env key: empty")
	}
	if strings.ContainsAny(k, "\n\r") {
		return fmt.Errorf("invalid env key %q: contains newline", k)
	}
	if strings.Contains(k, "=") {
		return fmt.Errorf("invalid env key %q: contains '='", k)
	}
	return nil
}

// validateEnvValue rejects values that would split a single .env line into
// multiple lines when written. The unquoted writer used by ApplyUpdates
// cannot represent embedded newlines safely, and surfacing the error to
// the caller is preferable to silently injecting unrelated keys.
func validateEnvValue(v string) error {
	if strings.ContainsAny(v, "\n\r") {
		return fmt.Errorf("invalid env value: contains newline")
	}
	return nil
}

// ReadKey returns the value of a single key from the .env file at path,
// or an empty string if the key is absent or the file cannot be read.
func ReadKey(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}

// ReadValues reads path once and returns all non-comment key/value pairs, with
// each value unquoted the same way ReadKey unquotes a single key. On a duplicate
// key the first occurrence wins, matching ReadKey, which returns its first
// match. Returns an empty (non-nil) map when the file is missing, so callers can
// range freely. Prefer this over repeated ReadKey calls when checking several
// keys from one file: ReadKey re-opens and rescans the whole file on every call.
func ReadValues(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if k = strings.TrimSpace(k); k != "" {
			if _, seen := out[k]; !seen {
				out[k] = strings.Trim(strings.TrimSpace(v), `"'`)
			}
		}
	}
	return out
}

// ReferencesContainer reports whether content references the lerd container
// hostname "lerd-<serviceName>" as a whole token, so bare "postgres" is not
// matched by a "lerd-postgres-18" reference (and vice versa). Commented-out
// lines are ignored so a disabled "#DB_HOST=lerd-mysql" doesn't keep a removed
// service's badge alive on the site page.
func ReferencesContainer(content, serviceName string) bool {
	needle := "lerd-" + serviceName
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if lineReferencesNeedle(stripInlineComment(line), needle) {
			return true
		}
	}
	return false
}

// stripInlineComment drops a trailing "# ..." comment from an .env line. A '#'
// only starts a comment when preceded by whitespace (dotenv convention), so a
// '#' inside a value (e.g. a password) is preserved.
func stripInlineComment(line string) string {
	for i := 1; i < len(line); i++ {
		if line[i] == '#' && (line[i-1] == ' ' || line[i-1] == '\t') {
			return line[:i]
		}
	}
	return line
}

// lineReferencesNeedle reports whether line contains needle as a whole token
// (the next byte, if any, is not part of a service name).
func lineReferencesNeedle(line, needle string) bool {
	for i := 0; ; {
		j := strings.Index(line[i:], needle)
		if j < 0 {
			return false
		}
		end := i + j + len(needle)
		if end >= len(line) || !isServiceNameByte(line[end]) {
			return true
		}
		i += j + 1
	}
}

// isServiceNameByte reports whether b can be part of a service name following
// the "lerd-" prefix (alphanumerics and '-', as in postgres-18 or mysql-5-7).
func isServiceNameByte(b byte) bool {
	return b == '-' ||
		(b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}

// ReadKeys returns all non-comment key names from the .env file at path,
// in the order they appear.
func ReadKeys(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var keys []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		k, _, _ := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

// UpdateAppURL sets the framework's URL key in envFile to scheme://domain.
// envFile and urlKey come from the framework definition (config.URLTargetFor),
// so Symfony's DEFAULT_URI in .env.local is written the same way Laravel's
// APP_URL in .env is. An empty urlKey means the framework holds its base URL
// somewhere other than the env file. Silently does nothing if the file is absent.
func UpdateAppURL(projectPath, envFile, urlKey, scheme, domain string) error {
	if urlKey == "" {
		return nil
	}
	if envFile == "" {
		envFile = ".env"
	}
	envPath := filepath.Join(projectPath, envFile)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil
	}
	return ApplyUpdates(envPath, map[string]string{
		urlKey: scheme + "://" + domain,
	})
}

// DomainScopedKeys lists every .env key that SyncPrimaryDomain will rewrite
// when the value is already present. Anything outside this set is considered
// a service/credentials/config concern and must NOT be touched by the
// automatic flows that run on link/init/UI upload.
//
// Exported so callers (tests, audits) can prove the scope is bounded.
var DomainScopedKeys = []string{
	"APP_URL",
	"ASSET_URL",
	"APP_DOMAIN",
	"VITE_APP_URL",
	"SESSION_DOMAIN",
	"SANCTUM_STATEFUL_DOMAINS",
	"VITE_REVERB_HOST",
	"VITE_REVERB_SCHEME",
	"VITE_REVERB_PORT",
	"REVERB_HOST",
	"REVERB_SCHEME",
	"REVERB_PORT",
}

// SyncPrimaryDomain updates the framework's URL key and VITE_REVERB_HOST/SCHEME/PORT
// in the project's env file to reflect the current primary domain and TLS state.
// envFile and urlKey are the framework's, resolved by config.URLTargetFor.
// Only keys that already exist in the .env are touched.
// Silently does nothing if no .env exists.
func SyncPrimaryDomain(projectPath, envFile, urlKey, domain string, secured bool) error {
	if urlKey == "" {
		return nil
	}
	if envFile == "" {
		envFile = ".env"
	}
	envPath := filepath.Join(projectPath, envFile)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil
	}

	keys, err := ReadKeys(envPath)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(keys))
	for _, k := range keys {
		present[k] = true
	}

	scheme := "http"
	port := "80"
	if secured {
		scheme = "https"
		port = "443"
	}
	url := scheme + "://" + domain

	// Each domain-scoped key gets a derived value from (scheme, domain, port).
	// Reverb keys mirror the broadcaster host/port pair; SANCTUM_STATEFUL_DOMAINS
	// is set to the bare host (Sanctum accepts a comma-separated list, but for
	// a freshly-uploaded project the single primary domain is the right default).
	derived := map[string]string{
		"APP_URL":                  url,
		"ASSET_URL":                url,
		"VITE_APP_URL":             url,
		"APP_DOMAIN":               domain,
		"SESSION_DOMAIN":           domain,
		"SANCTUM_STATEFUL_DOMAINS": domain,
		"VITE_REVERB_HOST":         domain,
		"VITE_REVERB_SCHEME":       scheme,
		"VITE_REVERB_PORT":         port,
		"REVERB_HOST":              domain,
		"REVERB_SCHEME":            scheme,
		"REVERB_PORT":              port,
	}

	updates := map[string]string{}
	if urlKey != "" && present[urlKey] {
		updates[urlKey] = url
	}
	for _, k := range DomainScopedKeys {
		if present[k] {
			updates[k] = derived[k]
		}
	}

	if len(updates) == 0 {
		return nil
	}
	return ApplyUpdates(envPath, updates)
}

// FrontendAPIBaseKeys lista as chaves de .env que apontam a base da API num
// projeto frontend (SPA). Só são reescritas se já existirem — mesma semântica
// e garantia de escopo de DomainScopedKeys. Nada fora deste set é tocado.
//
// Exportado para que callers (testes, auditorias) possam provar que o escopo
// é bounded. Projetos com outra convenção de chave devem adicioná-la aqui.
var FrontendAPIBaseKeys = []string{
	"URL_API",          // Quasar (gestao-clientes-spa)
	"VITE_API_URL",     // Vite genérico
	"VITE_APP_API_URL", // Vite/Vue convenção comum
}

// SyncFrontendAPIBase reescreve as chaves de FrontendAPIBaseKeys presentes no
// .env do projeto frontend para a origem unificada (scheme://domain, SEM /api —
// a SPA concatena seus próprios prefixos). Só toca chaves existentes,
// idempotente, best-effort se não houver .env.
func SyncFrontendAPIBase(projectPath, domain string, secured bool) error {
	envPath := filepath.Join(projectPath, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil
	}

	keys, err := ReadKeys(envPath)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(keys))
	for _, k := range keys {
		present[k] = true
	}

	scheme := "http"
	if secured {
		scheme = "https"
	}
	origin := scheme + "://" + domain

	updates := map[string]string{}
	for _, k := range FrontendAPIBaseKeys {
		if present[k] {
			updates[k] = origin
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return ApplyUpdates(envPath, updates)
}

// RevertFrontendAPIBase grava string vazia nas chaves de FrontendAPIBaseKeys
// presentes no .env do frontend, desfazendo o sync para a origem unificada.
// O lerd não conhece a URL de dev original, então um valor vazio (relativo,
// neutro) é o reset seguro. Só toca chaves existentes; no-op sem .env.
func RevertFrontendAPIBase(projectPath string) error {
	envPath := filepath.Join(projectPath, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil
	}
	keys, err := ReadKeys(envPath)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(keys))
	for _, k := range keys {
		present[k] = true
	}
	updates := map[string]string{}
	for _, k := range FrontendAPIBaseKeys {
		if present[k] {
			updates[k] = ""
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return ApplyUpdates(envPath, updates)
}

// isPHPSource reports whether a file's contents open with a PHP tag, the one
// shape the dotenv writer must never append to. A byte-order mark counts as
// leading whitespace: editors leave them behind and they do not make a settings
// file any less PHP.
func isPHPSource(content []byte) bool {
	body := bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	body = bytes.TrimLeft(body, " \t\r\n")
	return bytes.HasPrefix(body, []byte("<?php")) || bytes.HasPrefix(body, []byte("<?="))
}
