// Package structverify re-parses gsxmail's own rendered HTML with
// gotreesitter's HTML grammar and proves the output contract holds
// mechanically: zero parse-error nodes, balanced
// conditional comments, layout-table nesting under the cap, (for
// hardened-mode output) the role="presentation" / data-table split the
// contract states, a preheader div (when the rendered HTML has one) that
// carries its full suppression-style stack, and a dark-mode adaptive style
// layer (when present) that carries its own required selectors.
//
// This package is gsxmail's one deliberate, opt-in gotreesitter import:
// the core library render path never imports gotreesitter under any
// outcome. gsxmail's render path
// (renderhtml, doc, lower, gsxmail.go) never imports it; only tests and
// this package do. See structural_isolation_test.go at the module root
// for the build-graph proof, and structural_test.go for the pass run over
// every golden and the full block corpus.
package structverify

import (
	"fmt"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// Finding is one structural-verification result. Its Code values sit in
// the reserved EM131-EM134 catalog slots, but Finding is its own type —
// not lint.Diagnostic — so this package
// never has to import package lint (or vice versa) to report one.
type Finding struct {
	Code    string
	Message string
}

// String renders f the way a diagnostic line reads elsewhere in gsxmail:
// "CODE: message".
func (f Finding) String() string {
	return f.Code + ": " + f.Message
}

// MaxTableDepth is the layout-table nesting cap (EM132): Outlook renders
// deep table nesting unreliably past this depth.
const MaxTableDepth = 12

// htmlLang is the one Language this package ever constructs: gotreesitter
// ships ~206 grammars in its default embed, but structverify only ever
// asks for HTML.
//
// Binary-size measurement (taken 2026-08-16, gotreesitter v0.50.1):
// importing this package pulls the
// full default grammar registry into whatever binary links it, because
// grammars.HtmlLanguage() reads its blob through the package's default
// (untagged) blob source, which embeds all ~206 grammar blobs regardless
// of which one Language a caller actually asks for.
//
//   - Root test binary (`go test -c .`, this repo's own build, no
//     special tags — what plain `go test ./...` produces): 20,575,772 ->
//     40,473,378 bytes, a +19,897,606-byte (~19.0 MiB) delta. This is
//     test-only: it never ships.
//   - `cmd/gsxmail` CLI binary, this same commit, untagged: roughly 43.0
//     MiB (45,108,725 bytes); with the recommended `-tags 'grammar_subset
//     grammar_subset_html'`: roughly 24.4 MiB
//     (25,593,933 bytes) — see README's "Import from existing HTML" and
//     "The CLI" sections for the exact figures a specific build measures.
//     This package (`internal/structverify`) still never ships in that
//     binary: `cmd/gsxmail` reaches gotreesitter only through package
//     `importer`, for the `gsxmail import` verb — proved by
//     `TestGotreesitterIsolatedFromCorePath` and
//     `TestImporterIsolatedFromRenderPath` at the module root, which also
//     prove this package itself stays test-layer-only.
//   - Building this package's own importer (the module cache import, not
//     `m31labs.dev/gsxmail/importer`) with `-tags 'grammar_subset
//     grammar_subset_html'` cuts the root test binary delta to +373,552
//     bytes (~365 KiB) — under the dossier's ~5 MB gate with wide margin.
//     This test-layer package does not need the tags itself, since it
//     never ships; only `cmd/gsxmail`'s own build does.
var htmlLang = grammars.HtmlLanguage()

// parse re-parses html with the HTML grammar and returns its root node
// plus the source bytes Node.Text needs to slice back into.
func parse(html string) (*gts.Node, []byte, error) {
	src := []byte(html)
	parser := gts.NewParser(htmlLang)
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, nil, fmt.Errorf("structverify: parse: %w", err)
	}
	if tree == nil || tree.RootNode() == nil {
		return nil, nil, fmt.Errorf("structverify: parse produced no tree")
	}
	return tree.RootNode(), src, nil
}

// Verify re-parses html and checks the properties every gsxmail render
// must hold regardless of output contract (hardened or parity mode):
// no ERROR or erroneous_end_tag_name node anywhere in the tree (EM131 —
// the tree-sitter error-recovery grammar still produces a walkable tree
// for malformed HTML, so a clean parse is a real, mechanical guarantee,
// not just "did not panic"), every conditional comment that opens with
// "<!--[if" closes with the matching "<![endif]-->" (EM133), and no table
// nests deeper than MaxTableDepth (EM132).
func Verify(html string) ([]Finding, error) {
	root, src, err := parse(html)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	walkWellFormed(root, &findings)
	walkConditionalComments(root, src, &findings)
	walkTableDepth(root, src, 0, &findings)
	checkPreheaderStack(root, src, &findings)
	checkDarkStyleLayer(root, src, &findings)
	return findings, nil
}

// VerifyContract runs Verify's well-formedness checks, then additionally
// proves the hardened output contract's table split (EM134): every table
// carrying role="presentation" is a layout table and also carries
// border="0" cellpadding="0" cellspacing="0"; a table with a <th>
// descendant (a data table) never carries role="presentation".
//
// VerifyContract only applies to hardened-mode output. Parity mode
// (renderhtml.WriteOptions{Outlook:"off"}) intentionally keeps the
// original StatTable shape (role="presentation", <td> headers) and does not claim
// this contract; call Verify alone for parity-mode output.
func VerifyContract(html string) ([]Finding, error) {
	findings, err := Verify(html)
	if err != nil {
		return nil, err
	}
	root, src, err := parse(html)
	if err != nil {
		return nil, err
	}
	walkContract(root, src, &findings)
	return findings, nil
}

func walkWellFormed(n *gts.Node, findings *[]Finding) {
	if n == nil {
		return
	}
	typ := n.Type(htmlLang)
	if n.IsError() || typ == "erroneous_end_tag_name" || typ == "erroneous_end_tag" {
		p := n.StartPoint()
		*findings = append(*findings, Finding{
			Code: "EM131",
			Message: fmt.Sprintf(
				"emitted HTML does not re-parse cleanly: %s node at output line %d, column %d (structural verification pass)",
				typ, p.Row+1, p.Column+1),
		})
	}
	for i := 0; i < n.ChildCount(); i++ {
		walkWellFormed(n.Child(i), findings)
	}
}

// walkConditionalComments checks every "<!--[if ...]>" conditional
// comment's own text ends with the matching "<![endif]-->" marker
// (EM133). An HTML comment always lexes up to its first "-->" — if some
// other "-->" inside the intended conditional region closed the comment
// early (a writer bug that would silently corrupt the Outlook-only
// branch), the comment's text ends somewhere else instead.
func walkConditionalComments(n *gts.Node, src []byte, findings *[]Finding) {
	if n == nil {
		return
	}
	if n.Type(htmlLang) == "comment" {
		body := strings.TrimSpace(n.Text(src))
		if strings.HasPrefix(body, "<!--[if") && !strings.HasSuffix(body, "<![endif]-->") {
			p := n.StartPoint()
			*findings = append(*findings, Finding{
				Code: "EM133",
				Message: fmt.Sprintf(
					"conditional comment opened at output line %d has no matching <![endif]--> (structural verification pass)",
					p.Row+1),
			})
		}
	}
	for i := 0; i < n.ChildCount(); i++ {
		walkConditionalComments(n.Child(i), src, findings)
	}
}

func walkTableDepth(n *gts.Node, src []byte, depth int, findings *[]Finding) {
	if n == nil {
		return
	}
	nextDepth := depth
	if isElementNamed(n, src, "table") {
		nextDepth = depth + 1
		if nextDepth > MaxTableDepth {
			p := n.StartPoint()
			*findings = append(*findings, Finding{
				Code: "EM132",
				Message: fmt.Sprintf(
					"layout table nesting depth is %d; the cap is %d — Outlook renders deep nesting unreliably (output line %d, column %d)",
					nextDepth, MaxTableDepth, p.Row+1, p.Column+1),
			})
		}
	}
	for i := 0; i < n.ChildCount(); i++ {
		walkTableDepth(n.Child(i), src, nextDepth, findings)
	}
}

func walkContract(n *gts.Node, src []byte, findings *[]Finding) {
	if n == nil {
		return
	}
	if isElementNamed(n, src, "table") {
		checkTable(n, src, findings)
	}
	for i := 0; i < n.ChildCount(); i++ {
		walkContract(n.Child(i), src, findings)
	}
}

func checkTable(elem *gts.Node, src []byte, findings *[]Finding) {
	tag := startTag(elem)
	role, hasRole := attrValue(tag, src, "role")
	isPresentation := hasRole && role == "presentation"
	hasTH := elementHasDescendant(elem, src, "th")

	if isPresentation && hasTH {
		p := elem.StartPoint()
		*findings = append(*findings, Finding{
			Code: "EM134",
			Message: fmt.Sprintf(
				`data table must not carry role="presentation"; it holds tabular data (output line %d, column %d)`,
				p.Row+1, p.Column+1),
		})
	}
	if isPresentation {
		for _, want := range []string{"border", "cellpadding", "cellspacing"} {
			v, present := attrValue(tag, src, want)
			if !present || v != "0" {
				p := elem.StartPoint()
				*findings = append(*findings, Finding{
					Code: "EM134",
					Message: fmt.Sprintf(
						`layout table at output line %d, column %d carries role="presentation" but is missing %s="0"`,
						p.Row+1, p.Column+1, want),
				})
			}
		}
	}
}

// startTag returns n's start_tag or self_closing_tag child (an "element"
// node's opening tag), or nil if n is not an element or carries neither.
func startTag(n *gts.Node) *gts.Node {
	if n == nil {
		return nil
	}
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		switch c.Type(htmlLang) {
		case "start_tag", "self_closing_tag":
			return c
		}
	}
	return nil
}

// tagName reads a start_tag/self_closing_tag node's own tag_name child,
// lowercased (HTML tag names are case-insensitive).
func tagName(tag *gts.Node, src []byte) string {
	if tag == nil {
		return ""
	}
	for i := 0; i < tag.ChildCount(); i++ {
		c := tag.Child(i)
		if c.Type(htmlLang) == "tag_name" {
			return strings.ToLower(c.Text(src))
		}
	}
	return ""
}

// isElementNamed reports whether n is an "element" node whose start tag
// name equals name.
// isElementNamed reports whether n is an "element" node whose start tag
// name equals name, OR one of the HTML grammar's two special-cased raw-text
// element types ("style_element", "script_element") whose implied tag name
// ("style", "script") equals name — the grammar tags <style> and <script>
// distinctly from every other element (their content is raw text, not
// nested markup), but both still carry the same start_tag/end_tag shape
// startTag reads.
func isElementNamed(n *gts.Node, src []byte, name string) bool {
	switch n.Type(htmlLang) {
	case "element":
		return tagName(startTag(n), src) == name
	case "style_element":
		return name == "style"
	case "script_element":
		return name == "script"
	default:
		return false
	}
}

// attrValue returns the named attribute's value and whether it is present
// at all on tag (a start_tag or self_closing_tag node). A bare boolean
// attribute is present with an empty value.
func attrValue(tag *gts.Node, src []byte, name string) (value string, present bool) {
	if tag == nil {
		return "", false
	}
	for i := 0; i < tag.ChildCount(); i++ {
		c := tag.Child(i)
		if c.Type(htmlLang) != "attribute" {
			continue
		}
		var attrName, val string
		for j := 0; j < c.ChildCount(); j++ {
			gc := c.Child(j)
			switch gc.Type(htmlLang) {
			case "attribute_name":
				attrName = gc.Text(src)
			case "attribute_value":
				val = gc.Text(src)
			case "quoted_attribute_value":
				for k := 0; k < gc.ChildCount(); k++ {
					if gc.Child(k).Type(htmlLang) == "attribute_value" {
						val = gc.Child(k).Text(src)
					}
				}
			}
		}
		if strings.EqualFold(attrName, name) {
			return val, true
		}
	}
	return "", false
}

// elementHasDescendant reports whether n's subtree contains an element
// tagged target, without descending into a nested "table" element — a
// data-table check on the outer table must not see a header cell that
// actually belongs to an inner table.
func elementHasDescendant(n *gts.Node, src []byte, target string) bool {
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c.Type(htmlLang) == "element" {
			name := tagName(startTag(c), src)
			if name == target {
				return true
			}
			if name == "table" {
				continue
			}
		}
		if elementHasDescendant(c, src, target) {
			return true
		}
	}
	return false
}

// requiredPreheaderStyles are the six suppression declarations the pixel
// dossier's preheader contract requires (section 6.1, citing react-email's
// shipped Preview styles, R5).
var requiredPreheaderStyles = []string{
	"display:none", "overflow:hidden", "line-height:1px",
	"opacity:0", "max-height:0", "max-width:0",
}

// checkPreheaderStack implements EM173: when <body>'s first element
// child is a div whose style declares
// display:none — the shape renderhtml.writePreheader emits — every one of
// requiredPreheaderStyles must also be present. A template that renders
// with no preheader configured has no such div at all, so this check is a
// no-op for it (renderhtml.writePreheader writes nothing when Preheader is
// empty, in either output contract).
func checkPreheaderStack(root *gts.Node, src []byte, findings *[]Finding) {
	body := findElement(root, src, "body")
	if body == nil {
		return
	}
	first := firstElementChild(body)
	if first == nil || !isElementNamed(first, src, "div") {
		return
	}
	tag := startTag(first)
	style, present := attrValue(tag, src, "style")
	normalized := strings.ReplaceAll(style, " ", "")
	if !present || !strings.Contains(normalized, "display:none") {
		return // not a preheader-shaped div; nothing to check
	}
	var missing []string
	for _, want := range requiredPreheaderStyles {
		if !strings.Contains(normalized, strings.ReplaceAll(want, " ", "")) {
			missing = append(missing, want)
		}
	}
	if len(missing) > 0 {
		p := first.StartPoint()
		*findings = append(*findings, Finding{
			Code: "EM173",
			Message: fmt.Sprintf(
				"preheader div at output line %d, column %d is missing suppression style(s) %s",
				p.Row+1, p.Column+1, strings.Join(missing, ", ")),
		})
	}
}

// checkDarkStyleLayer implements EM174: a
// <style> block that declares @media (prefers-color-scheme:dark) — the
// "adaptive" strategy's own marker — must also carry both
// [data-ogsc]/[data-ogsb] Outlook-app inversion hooks and balanced
// braces. A "none" or "locked" style block never
// mentions prefers-color-scheme at all, so this check is a no-op for both.
func checkDarkStyleLayer(root *gts.Node, src []byte, findings *[]Finding) {
	style := findElement(root, src, "style")
	if style == nil {
		return
	}
	text := elementText(style, src)
	if !strings.Contains(text, "prefers-color-scheme") {
		return
	}
	var missing []string
	for _, want := range []string{"[data-ogsc]", "[data-ogsb]"} {
		if !strings.Contains(text, want) {
			missing = append(missing, want)
		}
	}
	if strings.Count(text, "{") != strings.Count(text, "}") {
		missing = append(missing, "balanced { }")
	}
	if len(missing) > 0 {
		*findings = append(*findings, Finding{
			Code:    "EM174",
			Message: fmt.Sprintf("adaptive dark-mode style layer is malformed: missing %s", strings.Join(missing, ", ")),
		})
	}
}

// findElement returns the first element named name anywhere in n's
// subtree (depth-first, including n itself), or nil.
func findElement(n *gts.Node, src []byte, name string) *gts.Node {
	if n == nil {
		return nil
	}
	if isElementNamed(n, src, name) {
		return n
	}
	for i := 0; i < n.ChildCount(); i++ {
		if found := findElement(n.Child(i), src, name); found != nil {
			return found
		}
	}
	return nil
}

// firstElementChild returns n's first direct child that is itself an
// "element" node, or nil.
func firstElementChild(n *gts.Node) *gts.Node {
	for i := 0; i < n.ChildCount(); i++ {
		if c := n.Child(i); c.Type(htmlLang) == "element" {
			return c
		}
	}
	return nil
}

// elementText concatenates every leaf node's own text within n, skipping
// its start/end tags — a grammar-agnostic way to read a <style> element's
// content regardless of exactly which leaf node type the HTML grammar
// gives raw CSS text.
func elementText(n *gts.Node, src []byte) string {
	var b strings.Builder
	var walk func(*gts.Node)
	walk = func(x *gts.Node) {
		switch x.Type(htmlLang) {
		case "start_tag", "end_tag", "self_closing_tag":
			return
		}
		if x.ChildCount() == 0 {
			b.WriteString(x.Text(src))
			return
		}
		for i := 0; i < x.ChildCount(); i++ {
			walk(x.Child(i))
		}
	}
	walk(n)
	return b.String()
}
