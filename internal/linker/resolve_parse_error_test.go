package linker

import (
	"strings"
	"testing"
)

// Resolve used to read the project config with `proj, _ :=`, so a .lerd.yaml
// that does not unmarshal was thrown away whole and the link carried on as if
// the project had declared nothing: name from the directory, no declared
// services, framework and PHP from detection. It reported success, which is how
// someone concludes multi-domain sites do not work (#1631).
func TestResolve_refusesAnUnparseableProjectConfig(t *testing.T) {
	dir := projectDir(t, "myapp", "- domains\n    - main\n    - subone\n")

	_, err := Resolve(dir, testConfig(), CLIPolicy("", false, nil))
	if err == nil {
		t.Fatal("a broken .lerd.yaml registered a site instead of failing")
	}
	if !strings.Contains(err.Error(), ".lerd.yaml") {
		t.Errorf("error = %q, want the file named so the fix is obvious", err)
	}
}

// A project with domains declared the documented way keeps working, so the
// error above is about the file being unreadable and nothing else.
func TestResolve_declaredDomainsSurviveTheParseCheck(t *testing.T) {
	dir := projectDir(t, "myapp", "domains:\n  - main\n  - subone\n  - subtwo.test\n")

	plan, err := Resolve(dir, testConfig(), CLIPolicy("", false, nil))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"main.test", "subone.test", "subtwo.test"}
	if !sliceEq(plan.Site.Domains, want) {
		t.Errorf("domains = %v, want %v", plan.Site.Domains, want)
	}
}
