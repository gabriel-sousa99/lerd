package desktopnotify

// Progress reports a long operation to the desktop while it runs. Off Linux, and
// on a session with no notification daemon, every method is a no-op so the
// operation still runs rather than being refused because nothing can draw it.
type Progress interface {
	// Step replaces the body with what is happening now.
	Step(text string)
	// Percent moves the bar. A total of zero leaves it alone, which is what the
	// stretch before the unit count is known looks like.
	Percent(done, total int)
	// Close takes the popup down.
	Close()
}

type noProgress struct{}

func (noProgress) Step(string)      {}
func (noProgress) Percent(int, int) {}
func (noProgress) Close()           {}
