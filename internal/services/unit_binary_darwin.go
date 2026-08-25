//go:build darwin

package services

// InstalledUnitBinary returns the program the installed agent runs, or "" when
// no plist of that name is on disk. Callers use it to tell an agent that still
// names a binary which is there from one left behind by a binary that moved.
func InstalledUnitBinary(name string) string {
	args, err := plistArgs(plistPath(name))
	if err != nil || len(args) == 0 {
		return ""
	}
	return args[0]
}
