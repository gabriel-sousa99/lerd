package envfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// This file handles the "php-array" env format: a PHP file that returns a
// nested array, as Magento's app/etc/env.php does. Keys are addressed by a
// dotted path (db.connection.default.host). The file is reparsed and reprinted
// rather than patched textually, which is what Magento's own
// DeploymentConfig\Writer does, so comments are not preserved by either.

type phpKind int

const (
	phpString phpKind = iota
	phpInt
	phpBool
	phpNull
	phpFloat
	phpArray

	// phpExpr is a value this reader cannot evaluate, held as the source text
	// that produced it: a function call, a constant, a concatenation. Only PHP
	// knows what it comes to, so lerd reports no value for the key and prints the
	// expression back untouched. Refusing the file instead would cost every other
	// key in it, and a framework whose defaults are written as calls (CakePHP's
	// app_local.php opens with filter_var(env('DEBUG', true), ...)) keeps its
	// whole configuration there.
	phpExpr
)

// phpValue is a parsed PHP value. Arrays keep their entry order so a rewrite
// does not reshuffle the file.
type phpValue struct {
	kind    phpKind
	str     string
	entries []phpEntry

	// start and end bracket the value's source, so a write can replace just that
	// span and leave the rest of the file alone. A value built in memory rather
	// than parsed has both zero.
	start int
	end   int
}

type phpEntry struct {
	key   string
	isInt bool // numeric (list) key, printed unquoted
	val   *phpValue
}

// ReadPhpArray parses a PHP file returning a nested array and flattens it to
// dotted keys. A file with no return statement yields an empty map, not an error.
func ReadPhpArray(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	root, err := parsePhpReturn(string(data))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if root == nil {
		return out, nil
	}
	flatten("", root, out)
	return out, nil
}

// ApplyPhpArrayUpdates sets each dotted key to its value, creating intermediate
// arrays as needed, and rewrites the file. A missing file (and its parent dirs)
// is created. An existing scalar keeps its type when the new value fits it.
func ApplyPhpArrayUpdates(path string, updates map[string]string) error {
	var root *phpValue
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if root, err = parsePhpReturn(string(data)); err != nil {
			return err
		}
	case !os.IsNotExist(err):
		return err
	}
	original := string(data)

	// Sort so the emitted file is deterministic when several new paths are added.
	keys := make([]string, 0, len(updates))
	for k := range updates {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// A key naming a node another key descends through cannot hold at the same
	// time as that key: the node is either the scalar or the array. The deeper
	// key is the one that makes the other's parent exist, so it wins, and the
	// shallower one is dropped rather than fought over the same span of the file.
	kept := keys[:0]
	for _, k := range keys {
		ancestor := false
		for _, other := range keys {
			if other != k && strings.HasPrefix(other, k+".") {
				ancestor = true
				break
			}
		}
		if !ancestor {
			kept = append(kept, k)
		}
	}
	keys = kept

	if root != nil && root.kind == phpArray {
		return writePhpArrayInPlace(path, original, root, keys, updates)
	}

	// No array to edit: the file is new, or holds something this reader does not
	// recognise as configuration. Print one.
	fresh := &phpValue{kind: phpArray}
	for _, k := range keys {
		setPath(fresh, strings.Split(k, "."), updates[k])
	}
	var b strings.Builder
	b.WriteString("<?php\nreturn ")
	printValue(&b, fresh, 0)
	b.WriteString(";\n")
	return writePhpArrayFile(path, original, b.String())
}

// writePhpArrayInPlace edits only the spans that change, leaving the rest of the
// file byte for byte. These files are read by people as much as by code:
// CakePHP's app_local.php is mostly guidance on what each key does, and opens
// with the `use function` import its own values depend on. Reprinting the parsed
// tree would return a valid array and a file that had lost all of it.
func writePhpArrayInPlace(path, original string, root *phpValue, keys []string, updates map[string]string) error {
	type edit struct {
		start, end int
		text       string
	}
	var edits []edit

	// Keys whose path does not exist yet are grafted onto the deepest array that
	// does, one insertion per array however many keys land in it, so two new keys
	// under the same new parent produce one entry rather than two of the same name.
	grafts := map[*phpValue]*phpValue{}
	var graftOrder []*phpValue
	replacements := map[*phpValue]*phpValue{}
	var replaceOrder []*phpValue

	type scalarEdit struct {
		node *phpValue
		text string
	}
	var scalars []scalarEdit

	for _, key := range keys {
		segs := strings.Split(key, ".")
		node, rest := descendPhpArray(root, segs)
		if len(rest) == 0 {
			scalars = append(scalars, scalarEdit{node,
				renderPhpValue(scalarValue(updates[key], node.kind), indentAt(original, node.start))})
			continue
		}
		if node.kind != phpArray {
			// Something that is not an array sits where one has to be. Replacing it
			// is the only way through, and every key reaching it shares the one
			// replacement: a second edit over the same span would splice over the first.
			replacement := replacements[node]
			if replacement == nil {
				replacement = &phpValue{kind: phpArray}
				replacements[node] = replacement
				replaceOrder = append(replaceOrder, node)
			}
			setPath(replacement, rest, updates[key])
			continue
		}
		graft := grafts[node]
		if graft == nil {
			graft = &phpValue{kind: phpArray}
			grafts[node] = graft
			graftOrder = append(graftOrder, node)
		}
		setPath(graft, rest, updates[key])
	}

	for _, s := range scalars {
		// A node other keys descend through is written as the array they need,
		// and that replacement covers this very span. Emitting both would put two
		// edits on it.
		if replacements[s.node] != nil {
			continue
		}
		edits = append(edits, edit{s.node.start, s.node.end, s.text})
	}

	for _, node := range replaceOrder {
		edits = append(edits, edit{node.start, node.end,
			renderPhpValue(replacements[node], indentAt(original, node.start))})
	}

	for _, node := range graftOrder {
		at, indent, ok := phpArrayInsertion(original, node)
		if !ok {
			// An array written on one line has no line to insert into, so the new
			// entries go inline before its closing bracket. Reprinting the node
			// whole would claim a span other edits may sit inside.
			var b strings.Builder
			for i, e := range grafts[node].entries {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString("'" + escapeSingle(e.key) + "' => ")
				printValue(&b, e.val, 0)
			}
			insertAt := node.end - 1
			edits = append(edits, edit{insertAt, insertAt, inlineSep(original, node.start, insertAt) + b.String()})
			continue
		}
		// The last entry may lack the trailing comma PHP needs between it and
		// what is inserted after it. Anything but whitespace between them (a
		// comment holding a comma, say) leaves this alone; the verify pass below
		// then refuses the write rather than guess.
		if n := len(node.entries); n > 0 {
			lastEnd := node.entries[n-1].val.end
			for lastEnd > 0 && (original[lastEnd-1] == ' ' || original[lastEnd-1] == '\t' || original[lastEnd-1] == '\n' || original[lastEnd-1] == '\r') {
				lastEnd--
			}
			if lastEnd <= at && !strings.Contains(original[lastEnd:at], ",") {
				edits = append(edits, edit{lastEnd, lastEnd, ","})
			}
		}
		var b strings.Builder
		for _, e := range grafts[node].entries {
			b.WriteString(indent + "'" + escapeSingle(e.key) + "' => ")
			printValue(&b, e.val, len(indent)/4)
			b.WriteString(",\n")
		}
		edits = append(edits, edit{at, at, b.String()})
	}

	// Applied back to front so each splice leaves the earlier offsets valid,
	// which holds only while every edit sits wholly before the one applied
	// before it. The edits are built never to overlap; one that does anyway
	// would be spliced against an offset that has already moved and cut the
	// file mid-expression, so it is an error, not a judgement call.
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := original
	bound := len(original)
	for _, e := range edits {
		if e.end > bound {
			return fmt.Errorf("refusing to rewrite %s: overlapping edits at offset %d", path, e.start)
		}
		out = out[:e.start] + e.text + out[e.end:]
		bound = e.start
	}
	if err := verifyPhpArrayRewrite(original, out, keys, updates); err != nil {
		return fmt.Errorf("refusing to rewrite %s: %w", path, err)
	}
	return writePhpArrayFile(path, original, out)
}

// inlineSep returns what separates an inline insertion from the entry before
// it: nothing straight after the opening bracket or an existing comma, a comma
// and space after an entry.
func inlineSep(src string, open, insertAt int) string {
	i := insertAt
	for i > open && (src[i-1] == ' ' || src[i-1] == '\t') {
		i--
	}
	if i <= open+1 || src[i-1] == ',' || src[i-1] == '[' || src[i-1] == '(' {
		return ""
	}
	return ", "
}

// verifyPhpArrayRewrite refuses a rewrite that damaged the file: the output
// must still parse, every update must read back as written, and every key the
// updates did not touch must still hold its old value. The writer has had more
// ways to be wrong than anyone predicted, and each reported success; whatever
// shape the next one takes, it becomes an error here instead of a corrupted
// config the application is left to discover.
func verifyPhpArrayRewrite(original, out string, keys []string, updates map[string]string) error {
	root, err := parsePhpReturn(out)
	if err != nil || root == nil {
		return fmt.Errorf("the rewrite no longer parses: %w", err)
	}
	got := map[string]string{}
	flatten("", root, got)
	for _, k := range keys {
		if got[k] != updates[k] {
			return fmt.Errorf("the rewrite lost %s: %q instead of %q", k, got[k], updates[k])
		}
	}
	origRoot, err := parsePhpReturn(original)
	if err != nil || origRoot == nil {
		return nil
	}
	was := map[string]string{}
	flatten("", origRoot, was)
	for k, v := range was {
		if updateTouches(k, keys) {
			continue
		}
		if got[k] != v {
			return fmt.Errorf("the rewrite changed %s unasked: %q instead of %q", k, got[k], v)
		}
	}
	return nil
}

// updateTouches reports whether an update key claims k: exactly, as one of its
// descendants, or as an ancestor a deeper update rebuilt on the way down.
func updateTouches(k string, keys []string) bool {
	for _, u := range keys {
		if u == k || strings.HasPrefix(k, u+".") || strings.HasPrefix(u, k+".") {
			return true
		}
	}
	return false
}

// descendPhpArray walks as far into the tree as the file already goes, returning
// the deepest node reached and the segments still to be created below it.
func descendPhpArray(root *phpValue, segs []string) (*phpValue, []string) {
	node := root
	for i, seg := range segs {
		if node.kind != phpArray {
			return node, segs[i:]
		}
		next := (*phpValue)(nil)
		for j := range node.entries {
			if node.entries[j].key == seg {
				next = node.entries[j].val
				break
			}
		}
		if next == nil {
			return node, segs[i:]
		}
		node = next
	}
	return node, nil
}

// phpArrayInsertion returns where a new entry goes in an array written across
// several lines, and the indentation to give it: the start of the line holding
// the closing bracket, so the entry becomes the last one and the bracket keeps
// its own line. Reports false for an array written on a single line.
func phpArrayInsertion(src string, arr *phpValue) (int, string, bool) {
	closer := arr.end - 1
	if closer <= arr.start {
		return 0, "", false
	}
	lineStart := strings.LastIndexByte(src[:closer], '\n') + 1
	if lineStart <= arr.start {
		return 0, "", false
	}
	closerIndent := src[lineStart:closer]
	if strings.TrimSpace(closerIndent) != "" {
		return 0, "", false
	}
	return lineStart, closerIndent + "    ", true
}

// indentAt returns the leading whitespace of the line offset sits on, so a
// replacement renders at the nesting the file already uses there.
func indentAt(src string, offset int) string {
	if offset > len(src) {
		return ""
	}
	lineStart := strings.LastIndexByte(src[:offset], '\n') + 1
	indent := src[lineStart:offset]
	if trimmed := strings.TrimLeft(indent, " \t"); trimmed != "" {
		return indent[:len(indent)-len(trimmed)]
	}
	return indent
}

// renderPhpValue prints a value as it should read at a given indentation.
func renderPhpValue(v *phpValue, indent string) string {
	var b strings.Builder
	printValue(&b, v, len(indent)/4)
	return b.String()
}

// clonePhpValue copies a parsed value deeply enough to graft onto without
// disturbing the offsets the edits are computed from.
func clonePhpValue(v *phpValue) *phpValue {
	out := &phpValue{kind: v.kind, str: v.str}
	for _, e := range v.entries {
		out.entries = append(out.entries, phpEntry{key: e.key, isInt: e.isInt, val: clonePhpValue(e.val)})
	}
	return out
}

// writePhpArrayFile persists a rewrite, skipping a write that changes nothing so
// an env sync doesn't churn a file the user has open. EnsureWorktreeEnv reaches
// here on every worktree sync.
func writePhpArrayFile(path, original, out string) error {
	if out == original {
		return nil
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return writeFile(path, []byte(out), 0o644)
}

func flatten(prefix string, v *phpValue, out map[string]string) {
	// An expression has no value until PHP runs it, so the key is left out
	// rather than reported as whatever its source text happens to read like.
	if v.kind == phpExpr {
		return
	}
	if v.kind != phpArray {
		out[prefix] = scalarString(v)
		return
	}
	for _, e := range v.entries {
		key := e.key
		if prefix != "" {
			key = prefix + "." + e.key
		}
		flatten(key, e.val, out)
	}
}

func scalarString(v *phpValue) string {
	if v.kind == phpNull {
		return ""
	}
	return v.str
}

// setPath walks (creating) the nested arrays named by segs and assigns value to
// the leaf, preserving the leaf's existing scalar type where the value fits.
func setPath(root *phpValue, segs []string, value string) {
	cur := root
	for i, seg := range segs {
		last := i == len(segs)-1
		idx := -1
		for j := range cur.entries {
			if cur.entries[j].key == seg {
				idx = j
				break
			}
		}
		if last {
			nv := scalarValue(value, existingKind(cur, idx))
			if idx >= 0 {
				cur.entries[idx].val = nv
			} else {
				cur.entries = append(cur.entries, phpEntry{key: seg, isInt: isIntKey(seg), val: nv})
			}
			return
		}
		if idx < 0 {
			child := &phpValue{kind: phpArray}
			cur.entries = append(cur.entries, phpEntry{key: seg, isInt: isIntKey(seg), val: child})
			cur = child
			continue
		}
		if cur.entries[idx].val.kind != phpArray {
			cur.entries[idx].val = &phpValue{kind: phpArray}
		}
		cur = cur.entries[idx].val
	}
}

func existingKind(parent *phpValue, idx int) phpKind {
	if idx < 0 {
		return phpString
	}
	return parent.entries[idx].val.kind
}

// scalarValue coerces a string update into the kind the file already used, so
// an int stays an int and a bool stays a bool. Anything that doesn't fit becomes
// a quoted string.
func scalarValue(value string, want phpKind) *phpValue {
	switch want {
	case phpInt:
		if _, err := strconv.Atoi(value); err == nil {
			return &phpValue{kind: phpInt, str: value}
		}
	case phpFloat:
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return &phpValue{kind: phpFloat, str: value}
		}
	case phpBool:
		if value == "true" || value == "false" {
			return &phpValue{kind: phpBool, str: value}
		}
	}
	return &phpValue{kind: phpString, str: value}
}

func isIntKey(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func printValue(b *strings.Builder, v *phpValue, depth int) {
	switch v.kind {
	case phpArray:
		if len(v.entries) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[\n")
		inner := strings.Repeat("    ", depth+1)
		for _, e := range v.entries {
			b.WriteString(inner)
			if e.isInt {
				b.WriteString(e.key)
			} else {
				b.WriteString("'" + escapeSingle(e.key) + "'")
			}
			b.WriteString(" => ")
			printValue(b, e.val, depth+1)
			b.WriteString(",\n")
		}
		b.WriteString(strings.Repeat("    ", depth) + "]")
	case phpInt, phpFloat, phpBool, phpExpr:
		b.WriteString(v.str)
	case phpNull:
		b.WriteString("null")
	default:
		b.WriteString("'" + escapeSingle(v.str) + "'")
	}
}

func escapeSingle(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `'`, `\'`)
}

// ── parser ──────────────────────────────────────────────────────────────────

type phpParser struct {
	src string
	pos int
}

// parsePhpReturn finds the top-level `return <value>;` and parses that value.
// Returns (nil, nil) when the file has no return statement.
func parsePhpReturn(src string) (*phpValue, error) {
	p := &phpParser{src: src}
	p.skipTrivia()
	for p.pos < len(p.src) {
		if p.hasWord("return") {
			p.pos += len("return")
			p.skipTrivia()
			return p.parseValue()
		}
		// Step over a string literal whole, so a `return` sitting inside one is
		// never mistaken for the statement. skipTrivia already ate comments.
		if c := p.src[p.pos]; c == '\'' || c == '"' {
			if _, err := p.parseString(); err != nil {
				return nil, err
			}
		} else {
			p.pos++
		}
		p.skipTrivia()
	}
	return nil, nil
}

// hasLiteral reports whether the cursor sits on one of PHP's keyword literals,
// which are case-insensitive: Drupal's settings.php writes FALSE, and reading
// that as an unsupported value would drop the whole statement.
func (p *phpParser) hasLiteral(w string) bool {
	if len(p.src)-p.pos < len(w) {
		return false
	}
	if !strings.EqualFold(p.src[p.pos:p.pos+len(w)], w) {
		return false
	}
	if p.pos > 0 && isIdentByte(p.src[p.pos-1]) {
		return false
	}
	after := p.pos + len(w)
	return after >= len(p.src) || !isIdentByte(p.src[after])
}

// hasWord reports whether the cursor sits on w as a standalone identifier.
func (p *phpParser) hasWord(w string) bool {
	if !strings.HasPrefix(p.src[p.pos:], w) {
		return false
	}
	if p.pos > 0 && isIdentByte(p.src[p.pos-1]) {
		return false
	}
	after := p.pos + len(w)
	return after >= len(p.src) || !isIdentByte(p.src[after])
}

func isIdentByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// skipTrivia consumes whitespace, the PHP open tag, and // # /* */ comments.
func (p *phpParser) skipTrivia() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.pos++
		case strings.HasPrefix(p.src[p.pos:], "<?php"):
			p.pos += len("<?php")
		case strings.HasPrefix(p.src[p.pos:], "//"), c == '#':
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
		case strings.HasPrefix(p.src[p.pos:], "/*"):
			if end := strings.Index(p.src[p.pos+2:], "*/"); end >= 0 {
				p.pos += 2 + end + 2
			} else {
				p.pos = len(p.src)
			}
		default:
			return
		}
	}
}

// parseValue records where the value it read begins and ends, so a writer can
// splice a replacement over exactly that span.
func (p *phpParser) parseValue() (*phpValue, error) {
	p.skipTrivia()
	start := p.pos
	v, err := p.parseValueAt()
	if err != nil {
		return nil, err
	}
	v.start, v.end = start, p.pos
	return v, nil
}

func (p *phpParser) parseValueAt() (*phpValue, error) {
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of file")
	}
	switch c := p.src[p.pos]; {
	case c == '[':
		p.pos++
		return p.parseArrayBody(']')
	case p.hasWord("array"):
		p.pos += len("array")
		p.skipTrivia()
		if p.pos >= len(p.src) || p.src[p.pos] != '(' {
			return nil, fmt.Errorf("expected ( after array at %d", p.pos)
		}
		p.pos++
		return p.parseArrayBody(')')
	case c == '\'' || c == '"':
		s, err := p.parseString()
		return &phpValue{kind: phpString, str: s}, err
	case p.hasLiteral("true"), p.hasLiteral("false"):
		w := "true"
		if p.hasLiteral("false") {
			w = "false"
		}
		p.pos += len(w)
		return &phpValue{kind: phpBool, str: w}, nil
	case p.hasLiteral("null"):
		p.pos += len("null")
		return &phpValue{kind: phpNull}, nil
	case c == '-' || c == '.' || c >= '0' && c <= '9':
		start := p.pos
		if c == '-' {
			p.pos++
		}
		kind := phpInt
		digits := 0
		for p.pos < len(p.src) {
			d := p.src[p.pos]
			if d == '.' || d == 'e' || d == 'E' || d == '+' || d == '-' {
				kind = phpFloat
			} else if d >= '0' && d <= '9' {
				digits++
			} else {
				break
			}
			p.pos++
		}
		// -PHP_INT_MAX is not the number "-": a sign with no digits behind it,
		// or a number running straight into an identifier, is an expression and
		// is kept whole.
		if digits == 0 || (p.pos < len(p.src) && isIdentByte(p.src[p.pos])) {
			p.pos = start
			return p.parseExpression()
		}
		return &phpValue{kind: kind, str: p.src[start:p.pos]}, nil
	}
	return p.parseExpression()
}

// parseExpression captures a value that is not a literal as the source text
// that produced it, so the rest of the file stays readable and writable around
// it. It ends where the value does: a comma or a closing bracket that belongs
// to the array holding it, neither of which it consumes. Brackets, strings and
// comments inside the expression are stepped over whole, so a comma within any
// of them does not end it early.
func (p *phpParser) parseExpression() (*phpValue, error) {
	start := p.pos
	depth := 0
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == '\'' || c == '"':
			if _, err := p.parseString(); err != nil {
				return nil, err
			}
			continue
		case strings.HasPrefix(p.src[p.pos:], "//"), c == '#',
			strings.HasPrefix(p.src[p.pos:], "/*"):
			p.skipTrivia()
			continue
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			if depth == 0 {
				return expressionValue(p.src[start:p.pos]), nil
			}
			depth--
		case c == ',' && depth == 0:
			return expressionValue(p.src[start:p.pos]), nil
		case c == ';' && depth == 0:
			return expressionValue(p.src[start:p.pos]), nil
		}
		p.pos++
	}
	return nil, fmt.Errorf("unterminated value at offset %d", start)
}

// expressionValue trims the trailing layout off captured source, so reprinting
// an array does not carry the old spacing before its comma into the new one.
func expressionValue(src string) *phpValue {
	return &phpValue{kind: phpExpr, str: strings.TrimRight(src, " \t\r\n")}
}

func (p *phpParser) parseArrayBody(closer byte) (*phpValue, error) {
	arr := &phpValue{kind: phpArray}
	next := 0
	for {
		p.skipTrivia()
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unterminated array")
		}
		if p.src[p.pos] == closer {
			p.pos++
			return arr, nil
		}
		first, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		p.skipTrivia()
		entry := phpEntry{}
		if strings.HasPrefix(p.src[p.pos:], "=>") {
			p.pos += 2
			val, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			entry.key = scalarString(first)
			entry.isInt = first.kind == phpInt
			entry.val = val
		} else {
			// Positional entry: synthesise the numeric key PHP would assign.
			entry.key = strconv.Itoa(next)
			entry.isInt = true
			entry.val = first
			next++
		}
		arr.entries = append(arr.entries, entry)
		p.skipTrivia()
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
		} else if p.pos >= len(p.src) || p.src[p.pos] != closer {
			// PHP requires the comma between entries; only the last may omit it.
			// Reading a file without them as if it parsed hides exactly the
			// corruption the writer's own guard exists to catch.
			return nil, fmt.Errorf("expected ',' or '%c' after array entry at offset %d", closer, p.pos)
		}
	}
}

func (p *phpParser) parseString() (string, error) {
	quote := p.src[p.pos]
	p.pos++
	var b strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == '\\' && p.pos+1 < len(p.src) {
			n := p.src[p.pos+1]
			switch {
			case n == quote || n == '\\':
				b.WriteByte(n)
				p.pos += 2
				continue
			case quote == '"' && n == 'n':
				b.WriteByte('\n')
				p.pos += 2
				continue
			case quote == '"' && n == 't':
				b.WriteByte('\t')
				p.pos += 2
				continue
			}
		}
		if c == quote {
			p.pos++
			return b.String(), nil
		}
		b.WriteByte(c)
		p.pos++
	}
	return "", fmt.Errorf("unterminated string")
}

// Values reads every key/value pair from an env file in the given format
// ("dotenv", "php-const", "php-array", "php-vars"). An unreadable file yields
// an empty (non-nil) map, so callers need no error path for a project whose env
// file isn't there yet. Prefer this over Reader when several keys are wanted, or
// when the absence of a key has to be told apart from an empty value.
func Values(path, format string) map[string]string {
	var (
		values map[string]string
		err    error
	)
	switch format {
	case "php-const":
		values, err = ReadPhpConst(path)
	case "php-array":
		values, err = ReadPhpArray(path)
	case "php-vars":
		values, err = ReadPhpVars(path)
	case "", FormatDotenv:
		return ReadValues(path)
	default:
		// A format from a newer store than this binary knows. Reading it as
		// dotenv would invent keys out of whatever the file happens to contain.
		return map[string]string{}
	}
	if err != nil {
		return map[string]string{}
	}
	return values
}

// Reader returns a key lookup for an env file in the given format ("dotenv",
// "php-const", "php-array", "php-vars"). An unreadable file yields a reader
// that returns empty strings, so callers need no error path for a missing env
// file.
func Reader(path, format string) func(key string) string {
	if format == "dotenv" || format == "" {
		return func(key string) string { return ReadKey(path, key) }
	}
	values := Values(path, format)
	return func(key string) string { return values[key] }
}
