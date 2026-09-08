//go:build !darwin && !linux

package desktopapp

// Install is a no-op off macOS: on Linux the application entry belongs to the
// separately installed Lerd Desktop flatpak, which owns its own .desktop file.
func Install() (string, error) { return "", nil }

// Remove is the matching no-op.
func Remove() error { return nil }

// Path reports where the launcher would live, empty when there is none.
func Path() string { return "" }
