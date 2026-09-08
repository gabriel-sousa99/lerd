package config

import "testing"

func TestApplyCommandPins_ChoiceOverridesDefault(t *testing.T) {
	cmds := []FrameworkCommand{
		{Name: "native:run", Pinned: true},
		{Name: "migrate"},
	}
	got := ApplyCommandPins(cmds, map[string]bool{"native:run": false, "migrate": true})
	if got[0].Pinned {
		t.Error("an explicit unpin must beat the definition's pinned default")
	}
	if !got[1].Pinned {
		t.Error("an explicit pin must beat the definition's default")
	}
	if cmds[0].Pinned != true || cmds[1].Pinned != false {
		t.Error("ApplyCommandPins must not mutate the input slice")
	}
}

func TestApplyCommandPins_CapsAtMax(t *testing.T) {
	cmds := []FrameworkCommand{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	got := ApplyCommandPins(cmds, map[string]bool{"a": true, "b": true, "c": true})
	if CountPinned(got) != MaxPinnedCommands {
		t.Fatalf("want %d pinned, got %d", MaxPinnedCommands, CountPinned(got))
	}
	if got[2].Pinned {
		t.Error("the cap should drop the last command, keeping resolution order")
	}
}

func TestApplyCommandPins_NoChoicesKeepsDefaults(t *testing.T) {
	cmds := []FrameworkCommand{{Name: "native:run", Pinned: true}, {Name: "migrate"}}
	got := ApplyCommandPins(cmds, nil)
	if !got[0].Pinned || got[1].Pinned {
		t.Errorf("defaults should carry through untouched: %+v", got)
	}
}

// TestSetSiteCommandPinned_RoundTrip pins that a personal pin choice survives the
// registry write and leaves the site's other fields alone.
func TestSetSiteCommandPinned_RoundTrip(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	seed := Site{Name: "a", Path: "/x", PHPVersion: "8.4", Paused: true}
	if err := SaveSites(&SiteRegistry{Sites: []Site{seed}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := SetSiteCommandPinned("a", "native:run", true); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := SetSiteCommandPinned("a", "migrate", false); err != nil {
		t.Fatalf("set: %v", err)
	}
	reg, err := LoadSites()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	s := reg.Sites[0]
	if !s.PinnedCommands["native:run"] || s.PinnedCommands["migrate"] {
		t.Errorf("PinnedCommands=%v, want native:run pinned and migrate unpinned", s.PinnedCommands)
	}
	if !s.Paused {
		t.Error("Paused should survive a pin write")
	}
	if err := SetSiteCommandPinned("missing", "migrate", true); err == nil {
		t.Error("want error for unknown site")
	}
}
