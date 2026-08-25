//go:build linux

package services

import (
	"os"
	"path/filepath"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// InstalledUnitBinary returns the program the installed unit runs, or "" when
// no unit of that name is on disk. Callers use it to tell a unit that still
// names a binary which is there from one left behind by a binary that moved.
func InstalledUnitBinary(name string) string {
	data, err := os.ReadFile(filepath.Join(config.SystemdUserDir(), name+".service"))
	if err != nil {
		return ""
	}
	return UnitExecBinary(string(data))
}
