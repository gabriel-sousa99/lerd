package serviceops

import (
	"strings"
	"testing"
)

// The six complaints a pg_dumpall replay makes against objects the target
// cluster cannot be without. Captured from a real postgis:18 replay into a
// fresh cluster with no user data in play.
const pgReplayNoise = `ERROR:  cannot drop a template database
ERROR:  current user cannot be dropped
ERROR:  role "postgres" already exists
ERROR:  database "template_postgis" already exists
ERROR:  schema "tiger" already exists
ERROR:  schema "topology" already exists
`

// isolatePresets points preset resolution at empty dirs. A store copy cached in
// the developer's own data dir is served above the embedded bundle, so without
// this these read whatever postgres.yaml that machine last fetched.
func isolatePresets(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestImportTallyDoesNotCountDeclaredExpectedErrors(t *testing.T) {
	isolatePresets(t)
	var tally ImportTally
	tally.Expect(expectedImportErrors("postgres", "import_all"))
	if _, err := tally.Stream().Write([]byte(pgReplayNoise)); err != nil {
		t.Fatalf("write: %v", err)
	}
	rep := tally.Report()
	if rep.Errors != 0 {
		t.Errorf("a clean replay must report no errors, got %d: %s", rep.Errors, rep.Summary())
	}
	if len(rep.Issues) != 0 {
		t.Errorf("expected complaints must not be listed, got %v", rep.Issues)
	}
}

// The filter has to stay narrow: a real failure in the same load is the whole
// reason the tally exists.
func TestImportTallyStillCountsRealErrorsAlongsideExpectedOnes(t *testing.T) {
	isolatePresets(t)
	var tally ImportTally
	tally.Expect(expectedImportErrors("postgres", "import_all"))
	if _, err := tally.Stream().Write([]byte(pgReplayNoise + "ERROR:  relation \"orders\" does not exist\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	rep := tally.Report()
	if rep.Errors != 1 {
		t.Fatalf("errors = %d, want the one real failure: %s", rep.Errors, rep.Summary())
	}
	if len(rep.Issues) != 1 || !strings.Contains(rep.Issues[0].Message, "orders") {
		t.Errorf("the real failure must survive the filter, got %v", rep.Issues)
	}
}

// Without the declaration the same lines are errors, which is what every other
// engine still gets and what makes this a preset decision rather than a rule
// about postgres compiled into Go.
func TestImportTallyCountsThemWithoutTheDeclaration(t *testing.T) {
	var tally ImportTally
	if _, err := tally.Stream().Write([]byte(pgReplayNoise)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := tally.Report().Errors; got != 6 {
		t.Errorf("errors = %d, want 6 when nothing is declared expected", got)
	}
}

func TestPostgresPresetDeclaresItsStructuralReplayErrors(t *testing.T) {
	isolatePresets(t)
	expected := expectedImportErrors("postgres", "import_all")
	if len(expected) == 0 {
		t.Fatal("the postgres preset must declare the errors its all-databases replay always makes")
	}
	for _, line := range strings.Split(strings.TrimSpace(pgReplayNoise), "\n") {
		var covered bool
		for _, pat := range expected {
			if strings.Contains(line, pat) {
				covered = true
				break
			}
		}
		if !covered {
			t.Errorf("no declared pattern covers %q", line)
		}
	}
}

// A single-database restore replays pg_dump into a database that was just
// dropped and recreated, so it has none of the cluster-level noise and must
// keep reporting everything it hits.
func TestPerDatabaseImportDeclaresNoExpectedErrors(t *testing.T) {
	isolatePresets(t)
	if got := expectedImportErrors("postgres", "import"); len(got) != 0 {
		t.Errorf("per-database import must not excuse anything, got %v", got)
	}
}

// The CLI tallies an import's output itself as it streams to the terminal, so
// it reads the declaration through this rather than through parseImportOutput.
func TestExpectedImportErrorsResolvesByScope(t *testing.T) {
	isolatePresets(t)
	if got := ExpectedImportErrors("postgres", true); len(got) == 0 {
		t.Error("all-databases scope must resolve the import_all declaration")
	}
	if got := ExpectedImportErrors("postgres", false); len(got) != 0 {
		t.Errorf("per-database scope must resolve the import declaration, got %v", got)
	}
}
