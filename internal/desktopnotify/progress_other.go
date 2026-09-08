//go:build !linux

package desktopnotify

// StartProgress is a no-op off Linux: macOS draws its progress from the launcher
// bundle, which can show a native window a notification cannot replace.
func StartProgress(_, _ string) Progress { return noProgress{} }
