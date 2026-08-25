package cli

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The Database select is single-choice precisely so a project cannot end up with
// two databases, and the Services multi-select filters its members out via the
// name set. A bundled default preset in a database family has to be in that set
// too: oracle-xe is one, and while it was missing the wizard offered it as a
// service you could tick while Database still said sqlite, writing both
// DB_CONNECTION values into .env.
func TestBuildDatabaseOptions_coversDefaultPresetsInDBFamilies(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	_, nameSet := buildDatabaseOptions(nil)

	for _, name := range config.DefaultPresetNames() {
		svc, err := config.DefaultPresetMeta(name)
		if err != nil || svc == nil {
			continue
		}
		family := dbFamilyOf(svc)
		if family == "" {
			continue
		}
		if !nameSet[name] {
			t.Errorf("default preset %q is in database family %q but is missing from the DB name set, so the Services multi-select would offer it alongside a separate Database choice", name, family)
		}
	}
}

// oracle-xe is the concrete case that regressed, so pin it by name as well.
func TestBuildDatabaseOptions_oracleXeIsADatabaseNotAService(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	options, nameSet := buildDatabaseOptions(nil)
	if !nameSet[("oracle-xe")] {
		t.Fatal("oracle-xe missing from the DB name set; it would leak into the Services multi-select")
	}
	var found bool
	for _, o := range options {
		if o.Value == "oracle-xe" {
			found = true
		}
	}
	if !found {
		t.Error("oracle-xe filtered out of Services but never offered as a Database option, leaving it unreachable in the wizard")
	}
}

// The exclusive-DB invariant the filter exists for: nothing offered as a service
// may also be a database.
func TestServiceOptions_excludeEveryDatabase(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	_, dbNameSet := buildDatabaseOptions(nil)
	for _, svc := range knownServices() {
		if dbNameSet[svc] {
			continue // correctly filtered
		}
		meta, err := config.DefaultPresetMeta(svc)
		if err != nil || meta == nil {
			continue
		}
		if fam := dbFamilyOf(meta); fam != "" {
			t.Errorf("service option %q resolves to database family %q; it must be filtered out of the Services list", svc, fam)
		}
	}
}
