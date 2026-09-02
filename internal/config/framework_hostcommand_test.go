package config

import "testing"

// A package whose command has to escape the container declares which arguments
// mean that and the binary to use. Without it `php artisan native:run` reaches
// lerd's shim, runs in the FPM container, and looks there for the Electron
// runtime and a display that only exist on the host (#1651).
func TestMatchHostCommand(t *testing.T) {
	fw := &Framework{HostCommands: []HostCommand{
		{Args: "artisan native:*", Binary: "vendor/nativephp/desktop/resources/build/php/php"},
	}}

	cases := []struct {
		name string
		argv []string
		want string
	}{
		{"the console command it declares", []string{"artisan", "native:run"}, "vendor/nativephp/desktop/resources/build/php/php"},
		{"trailing flags do not stop it matching", []string{"artisan", "native:build", "--no-interaction"}, "vendor/nativephp/desktop/resources/build/php/php"},
		{"another console command is left in the container", []string{"artisan", "migrate"}, ""},
		{"a bare script is left in the container", []string{"-v"}, ""},
		{"too few arguments to match the pattern", []string{"artisan"}, ""},
		{"nothing at all", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := MatchHostCommand(fw, tc.argv)
			if tc.want == "" {
				if ok {
					t.Fatalf("matched %q, want no match", got.Binary)
				}
				return
			}
			if !ok || got.Binary != tc.want {
				t.Fatalf("got (%q, %v), want %q", got.Binary, ok, tc.want)
			}
		})
	}
}

// A framework that declares nothing must never be asked to escape the
// container, and a nil one must not panic the passthrough.
func TestMatchHostCommand_NothingDeclared(t *testing.T) {
	if _, ok := MatchHostCommand(nil, []string{"artisan", "native:run"}); ok {
		t.Error("a nil framework declared nothing and must not match")
	}
	if _, ok := MatchHostCommand(&Framework{}, []string{"artisan", "native:run"}); ok {
		t.Error("a framework with no host_commands must not match")
	}
}

// The first declaration wins, so a package can be merged over a framework file
// without the older pattern shadowing it.
func TestMatchHostCommand_FirstMatchWins(t *testing.T) {
	fw := &Framework{HostCommands: []HostCommand{
		{Args: "artisan native:run", Binary: "first"},
		{Args: "artisan native:*", Binary: "second"},
	}}
	if got, _ := MatchHostCommand(fw, []string{"artisan", "native:run"}); got.Binary != "first" {
		t.Errorf("binary = %q, want first", got.Binary)
	}
	if got, _ := MatchHostCommand(fw, []string{"artisan", "native:build"}); got.Binary != "second" {
		t.Errorf("binary = %q, want second", got.Binary)
	}
}

// The binary and the command that installs it have to come from the same entry.
// A project carrying both nativephp packages merges two declarations, and the
// desktop installer does not put the mobile runtime on disk, so taking the
// binary from one and the hint from another sends the user to a command that
// cannot fix what they are looking at (#1651).
func TestMatchHostCommand_InstallCommandComesFromTheMatchedEntry(t *testing.T) {
	fw := &Framework{HostCommands: []HostCommand{
		{Args: "artisan native:jump", Binary: "vendor/nativephp/php-bin/bin/host/php", InstallCommand: "native:install-mobile"},
		{Args: "artisan native:*", Binary: "vendor/nativephp/electron/resources/js/resources/php/php", InstallCommand: "native:install"},
	}}

	got, ok := MatchHostCommand(fw, []string{"artisan", "native:jump"})
	if !ok || got.InstallCommand != "native:install-mobile" {
		t.Errorf("install command = %q, want native:install-mobile", got.InstallCommand)
	}
	if got.Binary != "vendor/nativephp/php-bin/bin/host/php" {
		t.Errorf("binary = %q, want the mobile runtime", got.Binary)
	}

	got, ok = MatchHostCommand(fw, []string{"artisan", "native:run"})
	if !ok || got.InstallCommand != "native:install" {
		t.Errorf("install command = %q, want native:install", got.InstallCommand)
	}
}

// A declaration published before install_command existed still matches, and
// carries no hint rather than a wrong one.
func TestMatchHostCommand_NoInstallCommandDeclared(t *testing.T) {
	fw := &Framework{HostCommands: []HostCommand{
		{Args: "artisan native:*", Binary: "vendor/bin/runtime"},
	}}
	got, ok := MatchHostCommand(fw, []string{"artisan", "native:run"})
	if !ok {
		t.Fatal("want a match")
	}
	if got.InstallCommand != "" {
		t.Errorf("install command = %q, want empty", got.InstallCommand)
	}
}
