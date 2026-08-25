package lifecycle

// TeardownOnLogout is false on Linux. There is no VM outliving the session to
// protect, and systemd already stops the container units as the user manager
// goes away, so a teardown here buys nothing and costs a real footgun: every
// SIGTERM the watcher gets is indistinguishable, so an ordinary
// `systemctl --user stop lerd-watcher` would stop every container, lerd-ui and
// lerd-dns with it.
const TeardownOnLogout = false

// BatchStopContainers is a no-op on Linux — systemd stops containers via unit
// deactivation so individual StopUnit calls are efficient and non-blocking.
func BatchStopContainers(_ []string) {}

// StopPodmanMachine is a no-op on Linux — Podman runs natively without a VM.
func StopPodmanMachine() {}
