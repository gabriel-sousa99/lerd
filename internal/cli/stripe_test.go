package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func findCmd(cmds []*cobra.Command, use string) *cobra.Command {
	for _, c := range cmds {
		if c.Use == use {
			return c
		}
	}
	return nil
}

func TestNewStripeCmds_ConfigCommandAndFlags(t *testing.T) {
	cmds := NewStripeCmds()

	cfg := findCmd(cmds, "stripe:config")
	if cfg == nil {
		t.Fatal("NewStripeCmds() must include a stripe:config command")
	}
	for _, f := range []string{"path", "secret-env-key"} {
		if cfg.Flags().Lookup(f) == nil {
			t.Errorf("stripe:config missing --%s flag", f)
		}
	}

	// The listener command must also expose the persistence flags so the path
	// can be set inline when starting.
	listen := findCmd(cmds, "stripe:listen")
	if listen == nil {
		t.Fatal("NewStripeCmds() must include a stripe:listen command")
	}
	for _, f := range []string{"path", "secret-env-key", "api-key"} {
		if listen.Flags().Lookup(f) == nil {
			t.Errorf("stripe:listen missing --%s flag", f)
		}
	}
}

// The stop command used to be registered at the root under the name
// stripe:listen, which left two root commands sharing that name and made
// `lerd stripe:listen` resolve to whichever cobra walked into first.
func TestNewStripeCmds_ListenStopIsASubcommand(t *testing.T) {
	root := &cobra.Command{Use: "lerd"}
	for _, c := range NewStripeCmds() {
		root.AddCommand(c)
	}

	listen := 0
	for _, c := range root.Commands() {
		if c.Name() == "stripe:listen" {
			listen++
		}
	}
	if listen != 1 {
		t.Fatalf("root has %d commands named stripe:listen, want 1", listen)
	}

	start, _, err := root.Find([]string{"stripe:listen"})
	if err != nil {
		t.Fatalf("finding stripe:listen: %v", err)
	}
	if start.Flags().Lookup("api-key") == nil {
		t.Error("lerd stripe:listen must resolve to the starter, which carries --api-key")
	}

	stop, _, err := root.Find([]string{"stripe:listen", "stop"})
	if err != nil {
		t.Fatalf("finding stripe:listen stop: %v", err)
	}
	if stop.Name() != "stop" {
		t.Errorf("lerd stripe:listen stop resolved to %q, want the stop subcommand", stop.Name())
	}
}
