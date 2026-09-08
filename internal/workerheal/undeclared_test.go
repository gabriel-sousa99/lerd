package workerheal

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func stubDeclared(t *testing.T, names map[string]bool, ok bool) {
	t.Helper()
	prev := declaredFn
	declaredFn = func(config.Site, *config.Framework) (map[string]bool, bool) { return names, ok }
	t.Cleanup(func() { declaredFn = prev })
}

// A worker retired upstream leaves its unit behind, failing on every tick. It
// answers to nothing, so there is no start that would heal it, and reporting it
// is what puts a banner on the dashboard for a worker the store stopped naming
// hours ago.
func TestDetect_undeclaredWorkerIsNotReported(t *testing.T) {
	stubEnv(t,
		[]string{"nativemob"}, nil,
		map[string]string{
			"lerd-jump-nativemob.service":  "failed",
			"lerd-queue-nativemob.service": "failed",
		},
		nil,
	)
	stubDeclared(t, map[string]bool{"queue": true}, true)

	got, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if names := unitNames(got); len(names) != 1 || names[0] != "lerd-queue-nativemob" {
		t.Errorf("got %v, want [lerd-queue-nativemob]", names)
	}
}

// An install that cannot resolve its store knows nothing about what is declared,
// and treating that silence as "nothing is declared" would drop every real
// failure off the dashboard at once.
func TestDetect_unresolvableDefinitionStillReports(t *testing.T) {
	stubEnv(t,
		[]string{"myapp"}, nil,
		map[string]string{"lerd-queue-myapp.service": "failed"},
		nil,
	)
	stubDeclared(t, nil, false)

	got, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if names := unitNames(got); len(names) != 1 || names[0] != "lerd-queue-myapp" {
		t.Errorf("got %v, want [lerd-queue-myapp]", names)
	}
}
