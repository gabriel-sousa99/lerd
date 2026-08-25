package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/editor"
)

// handleOpenEditor opens a file at a line in the host's editor for dashboard
// links such as a query's caller path. It requires dashboard-control authority,
// and paths are confined to the user's home directory.
func handleOpenEditor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !hasHostActionAuthority(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Path string `json:"path"`
		Line int    `json:"line"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	path := filepath.Clean(req.Path)
	if !filepath.IsAbs(path) {
		http.Error(w, "path must be absolute", http.StatusBadRequest)
		return
	}
	home, _ := os.UserHomeDir()
	if home == "" || !strings.HasPrefix(path, home+string(os.PathSeparator)) {
		http.Error(w, "path outside home", http.StatusForbidden)
		return
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	argv := editor.Command(path, req.Line)
	if len(argv) == 0 {
		http.Error(w, "no editor found; set `editor` in ~/.config/lerd/config.yaml", http.StatusInternalServerError)
		return
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Start(); err != nil {
		http.Error(w, fmt.Sprintf("launching editor: %v", err), http.StatusInternalServerError)
		return
	}
	go func() { _ = cmd.Wait() }() // reap; the editor detaches
	w.WriteHeader(http.StatusNoContent)
}
