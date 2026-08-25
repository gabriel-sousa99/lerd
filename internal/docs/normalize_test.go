package docs

import (
	"strings"
	"testing"
)

func TestNormalizeContainers(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "typed container without title",
			in:   "::: tip\nRun it twice.\n:::",
			want: []string{"> **Tip**", ">", "> Run it twice."},
		},
		{
			name: "typed container with title",
			in:   "::: warning Known limitation\nOnly loopback.\n:::",
			want: []string{"> **Warning: Known limitation**", ">", "> Only loopback."},
		},
		{
			name: "details container",
			in:   "::: details nginx fails\nCheck the log.\n:::",
			want: []string{"> **Details: nginx fails**", ">", "> Check the log."},
		},
		{
			name: "unknown container type keeps its own label",
			in:   "::: caution Careful\nMind the gap.\n:::",
			want: []string{"> **Caution: Careful**", ">", "> Mind the gap."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("Normalize() = %q, want it to contain %q", got, want)
				}
			}
			if strings.Contains(got, ":::") {
				t.Errorf("Normalize() left container markers behind: %q", got)
			}
		})
	}
}

func TestNormalizeContainerKeepsFencedCode(t *testing.T) {
	in := "::: details Long one\n```bash\nlerd start\n```\n:::"
	got := Normalize(in)

	for _, want := range []string{"> ```bash", "> lerd start"} {
		if !strings.Contains(got, want) {
			t.Errorf("Normalize() = %q, want it to contain %q", got, want)
		}
	}
}

func TestNormalizeCodeGroup(t *testing.T) {
	in := "::: code-group\n\n```bash [curl]\ncurl -fsSL https://lerd.sh | bash\n```\n\n```bash [wget]\nwget -qO- https://lerd.sh | bash\n```\n\n:::"
	got := Normalize(in)

	for _, want := range []string{"**curl**", "**wget**", "```bash", "curl -fsSL"} {
		if !strings.Contains(got, want) {
			t.Errorf("Normalize() = %q, want it to contain %q", got, want)
		}
	}
	if strings.Contains(got, "[curl]") {
		t.Errorf("Normalize() left the fence label in the info string: %q", got)
	}
	if strings.Contains(got, ":::") {
		t.Errorf("Normalize() left container markers behind: %q", got)
	}
}

func TestNormalizeCodeGroupInsideContainer(t *testing.T) {
	in := "::: details Both installers\n\n::: code-group\n\n```bash [curl]\ncurl\n```\n\n:::\n\n:::"
	got := Normalize(in)

	for _, want := range []string{"> **Details: Both installers**", "> **curl**", "> ```bash"} {
		if !strings.Contains(got, want) {
			t.Errorf("Normalize() = %q, want it to contain %q", got, want)
		}
	}
}

func TestNormalizeLeavesFencedContainersAlone(t *testing.T) {
	in := "```markdown\n::: tip\nnot a real container\n:::\n```"
	got := Normalize(in)

	if !strings.Contains(got, "::: tip") {
		t.Errorf("Normalize() rewrote a container inside a code fence: %q", got)
	}
}

func TestNormalizeInlineHTML(t *testing.T) {
	in := "| <code v-pre>{{domain}}</code> | The site domain |\n<div v-pre>\n\ntext <!-- markdownlint-disable -->\n\n</div>"
	got := Normalize(in)

	if !strings.Contains(got, "`{{domain}}`") {
		t.Errorf("Normalize() = %q, want the v-pre code span rewritten", got)
	}
	for _, unwanted := range []string{"<code", "<div", "</div>", "markdownlint"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("Normalize() = %q, want %q stripped", got, unwanted)
		}
	}
	if !strings.Contains(got, "| The site domain |") {
		t.Errorf("Normalize() dropped table content: %q", got)
	}
}

func TestNormalizePlainMarkdownIsUntouched(t *testing.T) {
	in := "# Title\n\nA paragraph with `code` and a [link](../usage/sites.md).\n\n- one\n- two\n"
	if got := Normalize(in); got != in {
		t.Errorf("Normalize() = %q, want it unchanged", got)
	}
}
