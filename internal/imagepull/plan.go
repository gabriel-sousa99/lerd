package imagepull

import (
	"fmt"
	"io"
	"sync"

	"github.com/gabriel-sousa99/lerd/internal/feedback"
)

// Item is one image a command is about to download or build.
type Item struct {
	Name   string // what the user recognises; falls back to Ref when empty
	Ref    string // registry reference to size up; empty for a pure local build
	Reason string // why this download is happening
	Build  bool   // a local build, which downloads its base image
	Bytes  int64  // estimated download size, 0 when unknown
}

// Pull describes an image lerd is about to fetch from a registry.
func Pull(ref, reason string) Item { return Item{Ref: ref, Reason: reason} }

// Build describes a local image build. baseRef is the image the build
// downloads before it can run, and may be empty when nothing is fetched.
func Build(name, baseRef, reason string) Item {
	return Item{Name: name, Ref: baseRef, Reason: reason, Build: true}
}

// Plan is everything a single command will download, disclosed in one block
// before the first byte moves.
type Plan []Item

func (i Item) label() string {
	if i.Name != "" {
		return i.Name
	}
	return i.Ref
}

// Fill looks up the download size of every item, concurrently. Items whose
// size cannot be read keep a zero and are disclosed without a number.
func (p Plan) Fill() Plan {
	var wg sync.WaitGroup
	for idx := range p {
		if p[idx].Ref == "" || p[idx].Bytes > 0 {
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if n, ok := Size(p[i].Ref); ok {
				p[i].Bytes = n
			}
		}(idx)
	}
	wg.Wait()
	return p
}

// Total is the sum of the known sizes in the plan.
func (p Plan) Total() int64 {
	var n int64
	for _, it := range p {
		n += it.Bytes
	}
	return n
}

// Report writes the disclosure block: what will be downloaded, how big it is
// and why. Nothing is written for an empty plan.
func (p Plan) Report(w io.Writer) {
	if len(p) == 0 {
		return
	}
	verb, tail := "will download", ""
	if DryRun() {
		verb, tail = "would download", "; nothing was downloaded"
	}
	noun := "images"
	if len(p) == 1 {
		noun = "image"
	}
	total := ""
	if n := p.Total(); n > 0 {
		total = fmt.Sprintf(" (~%s total)", Human(n))
	}
	on := feedback.ColorFor(w)
	fmt.Fprintf(w, "%s%s lerd %s %d %s%s%s\n", feedback.Prefix,
		feedback.DimIf(on, feedback.GlyphDownload), verb, len(p), noun, total, tail)

	sizes := make([]string, len(p))
	labelWidth, sizeWidth := 0, 0
	for i, it := range p {
		sizes[i] = "size unknown"
		if it.Bytes > 0 {
			sizes[i] = "~" + Human(it.Bytes)
		}
		labelWidth = max(labelWidth, len(it.label()))
		sizeWidth = max(sizeWidth, len(sizes[i]))
	}
	// Painted per cell rather than per line, so the columns are padded on the
	// plain text and stay aligned once the escapes are in.
	for i, it := range p {
		action := "pull"
		if it.Build {
			action = "rebuild"
		}
		line := fmt.Sprintf("%s   %s %s  %s", feedback.Prefix,
			feedback.DimIf(on, fmt.Sprintf("%-7s", action)),
			feedback.ValIf(on, fmt.Sprintf("%-*s", labelWidth, it.label())),
			fmt.Sprintf("%*s", sizeWidth, sizes[i]))
		if it.Reason != "" {
			line += "  " + feedback.DimIf(on, it.Reason)
		}
		fmt.Fprintln(w, line)
	}
}

// Note renders a size for appending to a one-line pull announcement. Empty
// when the registry did not answer, so the line reads normally without it.
func Note(bytes int64) string {
	if bytes <= 0 {
		return ""
	}
	return " (~" + Human(bytes) + ")"
}

// Human renders a byte count the way a download is normally quoted.
func Human(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit && exp < 3; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGT"[exp])
}
