package docs

import (
	"regexp"
	"strings"
)

var (
	containerRe   = regexp.MustCompile(`^:::+\s*([a-zA-Z][a-zA-Z-]*)\s*(.*)$`)
	fenceRe       = regexp.MustCompile("^(\\s*)(```+|~~~+)\\s*(.*)$")
	fenceLabelRe  = regexp.MustCompile(`^([a-zA-Z0-9_+-]*)\s*\[([^\]]+)\]\s*$`)
	vPreCodeRe    = regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`)
	htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	blockTagRe    = regexp.MustCompile(`(?i)^\s*</?(div|span|p)[^>]*>\s*$`)
)

var containerLabels = map[string]string{
	"info":    "Info",
	"tip":     "Tip",
	"warning": "Warning",
	"danger":  "Danger",
	"details": "Details",
}

// Normalize rewrites the VitePress-flavoured markdown under docs/ into portable
// markdown. Containers become blockquotes, code-group fence labels become a bold
// line above their block, and the inline HTML VitePress needs to escape template
// braces becomes a code span, so a plain CommonMark renderer shows the page
// instead of leaking the syntax. Both `lerd man` and the dashboard read it.
func Normalize(md string) string {
	lines := strings.Split(md, "\n")
	out := make([]string, 0, len(lines))

	var containers []string // open container kinds, innermost last
	quoted := false         // true while inside a callout container
	fence := ""             // open fence marker, empty outside a fence

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if fence != "" {
			if strings.HasPrefix(trimmed, fence) && strings.Trim(trimmed, string(fence[0])) == "" {
				fence = ""
			}
			out = append(out, quote(line, quoted))
			continue
		}

		if m := fenceRe.FindStringSubmatch(line); m != nil {
			fence = m[2]
			info := m[3]
			if label := fenceLabelRe.FindStringSubmatch(info); label != nil {
				out = append(out, quote("**"+label[2]+"**", quoted), quote("", quoted))
				info = label[1]
			}
			out = append(out, quote(m[1]+m[2]+info, quoted))
			continue
		}

		if strings.HasPrefix(trimmed, ":::") {
			if m := containerRe.FindStringSubmatch(trimmed); m != nil {
				kind := strings.ToLower(m[1])
				containers = append(containers, kind)
				if kind == "code-group" {
					out = append(out, quote("", quoted))
					continue
				}
				out = append(out, quote("", quoted))
				quoted = true
				out = append(out, "> **"+calloutLabel(kind, strings.TrimSpace(m[2]))+"**", ">")
				continue
			}
			if len(containers) > 0 {
				containers = containers[:len(containers)-1]
			}
			quoted = hasCallout(containers)
			out = append(out, quote("", quoted))
			continue
		}

		clean := cleanInlineHTML(line)
		if clean == "" && strings.TrimSpace(line) != "" {
			continue
		}
		out = append(out, quote(clean, quoted))
	}

	return strings.Join(out, "\n")
}

// quote prefixes a line for a blockquote when it sits inside a callout container.
func quote(line string, quoted bool) string {
	if !quoted {
		return line
	}
	if strings.TrimSpace(line) == "" {
		return ">"
	}
	return "> " + line
}

func hasCallout(containers []string) bool {
	for _, c := range containers {
		if c != "code-group" {
			return true
		}
	}
	return false
}

func calloutLabel(kind, title string) string {
	label, ok := containerLabels[kind]
	if !ok {
		label = toTitle(strings.ReplaceAll(kind, "-", " "))
	}
	if title != "" {
		return label + ": " + title
	}
	return label
}

// cleanInlineHTML strips the HTML VitePress needs and CommonMark does not: the
// v-pre wrappers that keep template braces literal, standalone block tags, and
// tooling comments. Anything left is returned as-is.
func cleanInlineHTML(line string) string {
	if !strings.Contains(line, "<") {
		return line
	}
	if blockTagRe.MatchString(line) {
		return ""
	}
	line = htmlCommentRe.ReplaceAllString(line, "")
	line = vPreCodeRe.ReplaceAllString(line, "`$1`")
	if strings.TrimSpace(line) == "" {
		return ""
	}
	return line
}
