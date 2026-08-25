package editor

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// isolate pins HOME and the XDG dirs to throwaway temp dirs so the config lookup
// never reads the real environment.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
}

func writeEditorConfig(t *testing.T, editor string) {
	t.Helper()
	cfgFile := config.GlobalConfigFile()
	if err := os.MkdirAll(filepath.Dir(cfgFile), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(cfgFile, []byte("editor: \""+editor+"\"\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestCommandConfigTemplate(t *testing.T) {
	tests := []struct {
		name   string
		editor string
		file   string
		line   int
		want   []string
	}{
		{
			name:   "file and line placeholders substituted",
			editor: "phpstorm --line {line} {file}",
			file:   "/home/u/app/Models/User.php",
			line:   42,
			want:   []string{"phpstorm", "--line", "42", "/home/u/app/Models/User.php"},
		},
		{
			name:   "only file placeholder substituted",
			editor: "myeditor {file}",
			file:   "/home/u/a.php",
			line:   9,
			want:   []string{"myeditor", "/home/u/a.php"},
		},
		{
			name:   "no placeholder appends the file",
			editor: "myeditor -w",
			file:   "/home/u/a.php",
			line:   7,
			want:   []string{"myeditor", "-w", "/home/u/a.php"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isolate(t)
			writeEditorConfig(t, tc.editor)
			got := Command(tc.file, tc.line)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Command() = %v, want %v", got, tc.want)
			}
		})
	}
}

// A directory has no line, so a configured template's {line} is dropped along
// with whatever introduced it: the flag in front of it, or the separator that
// joined it to {file}. Whole-word editors that take a bare path are unaffected.
func TestDirCommandConfigTemplate(t *testing.T) {
	tests := []struct {
		name   string
		editor string
		want   []string
	}{
		{
			name:   "flag and its line argument dropped",
			editor: "phpstorm --line {line} {file}",
			want:   []string{"phpstorm", "/home/u/site"},
		},
		{
			name:   "line joined to the file by a separator",
			editor: "code -g {file}:{line}",
			want:   []string{"code", "-g", "/home/u/site"},
		},
		{
			name:   "no line placeholder is a plain substitution",
			editor: "myeditor {file}",
			want:   []string{"myeditor", "/home/u/site"},
		},
		{
			name:   "no placeholder appends the directory",
			editor: "myeditor -w",
			want:   []string{"myeditor", "-w", "/home/u/site"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isolate(t)
			writeEditorConfig(t, tc.editor)
			got := DirCommand("/home/u/site")
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("DirCommand() = %v, want %v", got, tc.want)
			}
		})
	}
}

// Nothing configured and no known editor on PATH must report failure rather than
// fall back to the platform opener, which hands a directory to the file manager.
func TestDirCommandNoEditorFound(t *testing.T) {
	isolate(t)
	t.Setenv("PATH", t.TempDir())
	if got := DirCommand("/home/u/site"); got != nil {
		t.Fatalf("DirCommand() = %v, want nil with no editor available", got)
	}
}

// The detected editor opens the directory as a bare argument: the file variant's
// -g/--line forms address a location inside a file and don't apply here.
func TestDirCommandDetectedEditorTakesBarePath(t *testing.T) {
	isolate(t)
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "zed"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	want := []string{filepath.Join(bin, "zed"), "/home/u/site"}
	if got := DirCommand("/home/u/site"); !reflect.DeepEqual(got, want) {
		t.Fatalf("DirCommand() = %v, want %v", got, want)
	}
}
