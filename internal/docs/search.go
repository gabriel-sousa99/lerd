package docs

import "strings"

// Result is one page matching a search, with a snippet of the surrounding prose.
type Result struct {
	Title   string `json:"title"`
	Section string `json:"section"`
	Slug    string `json:"slug"`
	Route   string `json:"route"`
	Snippet string `json:"snippet"`
}

const snippetRadius = 90

// Search returns the pages matching the query, title matches first, each with a
// snippet of the text around the first hit. A blank query returns nothing.
func Search(pages []Page, query string, limit int) []Result {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	lower := strings.ToLower(query)

	var titleHits, bodyHits []Result
	for _, p := range pages {
		inTitle := strings.Contains(strings.ToLower(p.Title), lower)
		idx := strings.Index(strings.ToLower(p.content), lower)
		if !inTitle && idx < 0 {
			continue
		}
		r := Result{
			Title:   p.Title,
			Section: p.Section,
			Slug:    p.Slug,
			Route:   p.Route(),
			Snippet: snippet(p.content, idx, len(query)),
		}
		if inTitle {
			titleHits = append(titleHits, r)
		} else {
			bodyHits = append(bodyHits, r)
		}
	}

	results := append(titleHits, bodyHits...)
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// snippet returns the plain text around a match, widened to whole words.
func snippet(content string, idx, matchLen int) string {
	if idx < 0 {
		return ""
	}
	start := max(idx-snippetRadius, 0)
	end := min(idx+matchLen+snippetRadius, len(content))
	if sp := strings.IndexAny(content[start:idx], " \n"); sp >= 0 && start > 0 {
		start += sp + 1
	}
	if sp := strings.LastIndexAny(content[idx+matchLen:end], " \n"); sp >= 0 && end < len(content) {
		end = idx + matchLen + sp
	}

	text := strings.Join(strings.Fields(content[start:end]), " ")
	text = strings.NewReplacer("`", "", "*", "", "#", "").Replace(text)
	if start > 0 {
		text = "… " + text
	}
	if end < len(content) {
		text += " …"
	}
	return text
}
