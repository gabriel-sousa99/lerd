package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidFrameworkName(t *testing.T) {
	valid := []string{"laravel", "symfony", "wordpress", "a", "foo.bar", "foo-bar_baz", "x10"}
	for _, s := range valid {
		if !ValidFrameworkName(s) {
			t.Errorf("ValidFrameworkName(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",                                   // empty
		"../../../../home/user/.config/evil", // traversal
		"..",                                 // traversal
		"foo/bar",                            // path separator
		"/etc/passwd",                        // absolute-ish
		"foo\x00bar",                         // NUL
		"Laravel",                            // uppercase
		"foo bar",                            // space
		".hidden",                            // leading dot (not [a-z0-9])
		"-leading",                           // leading dash
	}
	for _, s := range invalid {
		if ValidFrameworkName(s) {
			t.Errorf("ValidFrameworkName(%q) = true, want false", s)
		}
	}
}

func TestValidFrameworkVersion(t *testing.T) {
	// empty version is allowed (unversioned framework)
	if !ValidFrameworkVersion("") {
		t.Error("ValidFrameworkVersion(\"\") = false, want true")
	}
	for _, s := range []string{"11", "10.x", "7", "8.2"} {
		if !ValidFrameworkVersion(s) {
			t.Errorf("ValidFrameworkVersion(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"../evil", "1/2", "v\x00"} {
		if ValidFrameworkVersion(s) {
			t.Errorf("ValidFrameworkVersion(%q) = true, want false", s)
		}
	}
}

func TestSaveStoreFramework_RejectsTraversalName(t *testing.T) {
	setConfigDir(t)

	dir := StoreFrameworksDir()
	// Marker file outside the store dir that a traversal payload would target.
	parent := filepath.Dir(dir)
	evilTarget := filepath.Join(parent, "evil.yaml")

	fw := &Framework{Name: "../evil"}
	if err := SaveStoreFramework(fw); err == nil {
		t.Fatal("expected SaveStoreFramework to reject traversal name, got nil error")
	}
	if _, err := os.Stat(evilTarget); !os.IsNotExist(err) {
		t.Fatalf("traversal write escaped store dir: %s exists", evilTarget)
	}
}

func TestSaveStoreFramework_RejectsTraversalVersion(t *testing.T) {
	setConfigDir(t)

	fw := &Framework{Name: "laravel", Version: "../../evil"}
	if err := SaveStoreFramework(fw); err == nil {
		t.Fatal("expected SaveStoreFramework to reject traversal version, got nil error")
	}
}

func TestSaveStoreFramework_RejectsNUL(t *testing.T) {
	setConfigDir(t)

	if err := SaveStoreFramework(&Framework{Name: "foo\x00bar"}); err == nil {
		t.Fatal("expected SaveStoreFramework to reject NUL in name")
	}
}

func TestSaveStoreFramework_AcceptsLegitimate(t *testing.T) {
	setConfigDir(t)

	cases := []struct {
		fw   *Framework
		file string
	}{
		{&Framework{Name: "laravel"}, "laravel.yaml"},
		{&Framework{Name: "symfony", Version: "7"}, "symfony@7.yaml"},
		{&Framework{Name: "laravel", Version: "11"}, "laravel@11.yaml"},
		{&Framework{Name: "wordpress", Version: "10.x"}, "wordpress@10.x.yaml"},
	}
	for _, c := range cases {
		if err := SaveStoreFramework(c.fw); err != nil {
			t.Fatalf("SaveStoreFramework(%+v): unexpected error: %v", c.fw, err)
		}
		path := filepath.Join(StoreFrameworksDir(), c.file)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to be written, stat error: %v", path, err)
		}
	}
}
