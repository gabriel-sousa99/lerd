//go:build linux

package desktopnotify

import (
	"sync"

	"github.com/godbus/dbus/v5"
)

// StartProgress opens a notification that is rewritten in place as the work
// proceeds. It goes over the session bus lerd already speaks for every other
// notification, so a progress window costs no tool the desktop may not have.
//
// The "value" hint is the freedesktop convention for a progress bar; a daemon
// that does not render one still shows the summary and the step, which is the
// part that says the click did something.
func StartProgress(summary, body string) Progress {
	if !Supported() {
		return noProgress{}
	}
	p := &busProgress{summary: summary, body: body}
	p.push()
	if p.id == 0 {
		return noProgress{}
	}
	return p
}

type busProgress struct {
	mu      sync.Mutex
	id      uint32
	summary string
	body    string
	percent int
	closed  bool
}

func (p *busProgress) Step(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.body = text
	p.push()
}

func (p *busProgress) Percent(done, total int) {
	if total <= 0 {
		return
	}
	if done > total {
		done = total
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.percent = done * 100 / total
	p.push()
}

func (p *busProgress) Close() {
	p.mu.Lock()
	id := p.id
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	if id == 0 {
		return
	}
	conn, err := sessionBus()
	if err != nil {
		return
	}
	call(conn.Object(notifyDest, notifyPath), notifyDest+".CloseNotification", id)
}

// push sends or rewrites the popup. Passing the id back as replaces_id is what
// keeps one notification updating rather than stacking one per unit.
func (p *busProgress) push() {
	conn, err := sessionBus()
	if err != nil {
		return
	}
	hints := map[string]dbus.Variant{
		"urgency":     dbus.MakeVariant(byte(UrgencyLow)),
		"value":       dbus.MakeVariant(int32(p.percent)),
		"transient":   dbus.MakeVariant(true),
		"synchronous": dbus.MakeVariant("lerd-start"),
	}
	c := call(conn.Object(notifyDest, notifyPath), notifyDest+".Notify",
		"Lerd", p.id, IconPath(), p.summary, p.body,
		[]string{}, hints, int32(0))
	if c.Err != nil {
		return
	}
	var id uint32
	if err := c.Store(&id); err == nil {
		p.id = id
	}
}
