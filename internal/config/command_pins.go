package config

// MaxPinnedCommands bounds how many commands a site draws as buttons on its
// control row. The row already folds its secondary actions into a menu on a
// narrow panel, so an unbounded set would land on that fold first.
const MaxPinnedCommands = 2

// ApplyCommandPins settles each command's Pinned flag: the user's choice for
// the site wins over the default its definition declared, and no more than
// MaxPinnedCommands stay pinned, in the order the commands resolve.
func ApplyCommandPins(cmds []FrameworkCommand, choices map[string]bool) []FrameworkCommand {
	out := make([]FrameworkCommand, len(cmds))
	copy(out, cmds)
	kept := 0
	for i := range out {
		if v, ok := choices[out[i].Name]; ok {
			out[i].Pinned = v
		}
		if !out[i].Pinned {
			continue
		}
		if kept >= MaxPinnedCommands {
			out[i].Pinned = false
			continue
		}
		kept++
	}
	return out
}

// CountPinned reports how many of the commands are pinned.
func CountPinned(cmds []FrameworkCommand) int {
	n := 0
	for _, c := range cmds {
		if c.Pinned {
			n++
		}
	}
	return n
}
