package store

import (
	"os"
	"testing"
)

// TestMain points the XDG roots at a throwaway directory for the whole package,
// so no test can write the developer's real framework store. Fetching is the
// business of this package and several of its entry points cache what they pull,
// including ones that only read: resolving a latest version refreshes the index.
// A test that forgets to isolate would otherwise overwrite a real catalogue with
// a two-framework fixture, which the tests themselves cannot notice. Tests that
// want their own directory still set these with t.Setenv.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "lerd-store-tests")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_DATA_HOME", tmp)   //nolint:errcheck
	os.Setenv("XDG_CONFIG_HOME", tmp) //nolint:errcheck
	code := m.Run()
	os.RemoveAll(tmp) //nolint:errcheck
	os.Exit(code)
}
