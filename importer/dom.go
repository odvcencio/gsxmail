package importer

import (
	"fmt"
	"html"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// htmlLang is the one gotreesitter Language this package ever asks for
// (structverify's own doc comment states the same restraint). Since
// gsxmail never needs any other grammar, `-tags 'grammar_subset
// grammar_subset_html'` is the CLI's recommended, documented default
// build (README's "The CLI" and "Import from existing HTML" sections,
// m6): the untagged build still works — gotreesitter's own default is
// "embed every grammar" — but it pays roughly 19 MiB for languages this
// package never asks for. This package itself carries no build tag, so a
// plain `go test ./...` still links every embedded grammar, exactly like
// internal/structverify already does; binsize_test.go is what
// actually measures and gates the recommended tagged CLI build's size.
var htmlLang = grammars.HtmlLanguage()

// node is one importer-owned DOM node: a normalized view of a
// gotreesitter HTML parse tree that the mapper walks without touching the
// grammar's own node-type strings again. Building this tree once,
// up front, is what makes gotreesitter's error tolerance actually usable
// here — a malformed source (an unclosed <td>, a stray "-->") still
// yields a walkable node tree, and every downstream heuristic in
// mapper.go only ever sees this shape.
type node struct {
	tag    string // lowercased tag name; "" for a text or comment node
	isText bool
	text   string // decoded text content (isText only)

	isComment bool
	comment   string // the comment's own trimmed interior text (isComment only)

	// errorish marks a node the grammar itself flagged: an ERROR node, or
	// an erroneous_end_tag(_name) — gotreesitter's error-recovery grammar
	// keeps walking past these rather than failing the parse (pixel
	// dossier section 7.1), and so does domify; the mapper reports an
	// errorish node as unmapped rather than guessing its shape.
	errorish bool

	attrs    []attr
	children []node
}

type attr struct {
	name  string
	value string
}

// attr returns name's value on n (case-insensitive) and whether it was
// present at all. A bare boolean attribute (no "=value") is present with
// an empty value.
func (n *node) attr(name string) (string, bool) {
	for _, a := range n.attrs {
		if strings.EqualFold(a.name, name) {
			return a.value, true
		}
	}
	return "", false
}

// style returns n's own "style" attribute, parsed into name: value
// declarations, lowercased property names, in source order. A missing
// style attribute returns nil.
func (n *node) style() []styleDecl {
	raw, ok := n.attr("style")
	if !ok {
		return nil
	}
	return parseStyleDecls(raw)
}

// styleValue looks up prop (already lowercase) in n's style attribute.
func (n *node) styleValue(prop string) (string, bool) {
	for _, d := range n.style() {
		if d.prop == prop {
			return d.value, true
		}
	}
	return "", false
}

// hasClass reports whether n's "class" attribute contains name as one of
// its space-separated tokens.
func (n *node) hasClass(name string) bool {
	raw, ok := n.attr("class")
	if !ok {
		return false
	}
	for _, c := range strings.Fields(raw) {
		if c == name {
			return true
		}
	}
	return false
}

// text collects every text-node descendant of n, in document order,
// joined with single spaces and whitespace-collapsed — n's own "flattened
// inner text," the shape most fingerprint checks and every derived props
// value need. It stops at (but does not skip past) nested block elements
// nowhere: text is deliberately non-structural, so it always descends
// fully.
func (n *node) innerText() string {
	var b strings.Builder
	var walk func(*node)
	walk = func(x *node) {
		if x.isText {
			b.WriteString(x.text)
			return
		}
		if x.isComment {
			return
		}
		for i := range x.children {
			walk(&x.children[i])
		}
	}
	walk(n)
	return collapseSpace(b.String())
}

// collapseSpace trims and collapses every run of whitespace in s to one
// space — HTML's own "insignificant whitespace" rule, needed because the
// source is pretty-printed with newlines and indentation domify's text
// nodes carry through verbatim.
func collapseSpace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// elements returns n's direct element children (skipping text, comment,
// and errorish nodes), transparently flattening a "tbody"/"thead"/"tfoot"
// wrapper (real emails almost never author one, but a browser-normalized
// or hand-written source sometimes carries one; gsxmail's own <tr> rows
// never do) so a caller walking a <table>'s own rows never has to special-
// case it.
func (n *node) elements() []*node {
	var out []*node
	for i := range n.children {
		c := &n.children[i]
		if c.isText || c.isComment {
			continue
		}
		if c.tag == "tbody" || c.tag == "thead" || c.tag == "tfoot" {
			out = append(out, c.elements()...)
			continue
		}
		out = append(out, c)
	}
	return out
}

// findAll returns every descendant of n (n included) whose tag equals
// name, in document order, including a match nested inside another match
// (a <table> inside a <table>'s own cell reports both) — callers that
// need to stop at the first same-tag boundary instead (a check on one
// table's own rows that must not see an unrelated nested table's cells)
// use findAllShallow, not this function.
func findAll(n *node, name string) []*node {
	var out []*node
	var walk func(*node)
	walk = func(x *node) {
		if x.tag == name {
			out = append(out, x)
		}
		for i := range x.children {
			walk(&x.children[i])
		}
	}
	walk(n)
	return out
}

// findFirst returns the first descendant of n (n included, document
// order) whose tag equals name, or nil.
func findFirst(n *node, name string) *node {
	if n.tag == name {
		return n
	}
	for i := range n.children {
		if f := findFirst(&n.children[i], name); f != nil {
			return f
		}
	}
	return nil
}

// parseHTML parses src with gotreesitter's HTML grammar and returns the
// domified tree. It never fails on malformed markup — the grammar's own
// error-recovery contract (pixel dossier section 7.1) — only on an empty
// or unparseable byte stream gotreesitter itself refuses outright.
func parseHTML(src []byte) (*node, error) {
	parser := gts.NewParser(htmlLang)
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("importer: parse: %w", err)
	}
	if tree == nil || tree.RootNode() == nil {
		return nil, fmt.Errorf("importer: parse produced no tree")
	}
	root := domify(tree.RootNode(), src)
	return root, nil
}

// domify converts one gotreesitter node (and its subtree) to the
// importer's own node shape.
func domify(n *gts.Node, src []byte) *node {
	typ := n.Type(htmlLang)
	switch typ {
	case "text", "raw_text":
		return &node{isText: true, text: html.UnescapeString(n.Text(src))}
	case "comment":
		body := n.Text(src)
		body = strings.TrimPrefix(body, "<!--")
		body = strings.TrimSuffix(body, "-->")
		return &node{isComment: true, comment: strings.TrimSpace(body)}
	case "doctype":
		return &node{tag: "!doctype"}
	case "erroneous_end_tag_name", "erroneous_end_tag":
		return &node{errorish: true, text: n.Text(src)}
	}

	out := &node{errorish: n.IsError()}
	switch typ {
	case "element":
		tag, attrs := domifyOpenTag(n, src)
		out.tag = tag
		out.attrs = attrs
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			ct := c.Type(htmlLang)
			if ct == "start_tag" || ct == "end_tag" || ct == "self_closing_tag" {
				continue
			}
			out.children = append(out.children, *domify(c, src))
		}
	case "style_element", "script_element":
		out.tag = strings.TrimPrefix(typ, "")
		if typ == "style_element" {
			out.tag = "style"
		} else {
			out.tag = "script"
		}
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			ct := c.Type(htmlLang)
			if ct == "start_tag" || ct == "end_tag" {
				continue
			}
			out.children = append(out.children, *domify(c, src))
		}
	case "document", "fragment":
		for i := 0; i < n.ChildCount(); i++ {
			out.children = append(out.children, *domify(n.Child(i), src))
		}
	default:
		// An unrecognized node type (a future grammar addition, or a
		// grammar-internal node this package has no opinion about): treat
		// it as an opaque container and keep walking its children so a
		// parse never silently drops content.
		for i := 0; i < n.ChildCount(); i++ {
			out.children = append(out.children, *domify(n.Child(i), src))
		}
	}
	return out
}

// domifyOpenTag reads an "element" node's start_tag (or self_closing_tag)
// child for its lowercased tag name and attribute list.
func domifyOpenTag(n *gts.Node, src []byte) (string, []attr) {
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		switch c.Type(htmlLang) {
		case "start_tag", "self_closing_tag":
			return tagAndAttrs(c, src)
		}
	}
	return "", nil
}

func tagAndAttrs(tag *gts.Node, src []byte) (string, []attr) {
	name := ""
	var attrs []attr
	for i := 0; i < tag.ChildCount(); i++ {
		c := tag.Child(i)
		switch c.Type(htmlLang) {
		case "tag_name":
			name = strings.ToLower(c.Text(src))
		case "attribute":
			a := domifyAttribute(c, src)
			attrs = append(attrs, a)
		}
	}
	return name, attrs
}

func domifyAttribute(n *gts.Node, src []byte) attr {
	var a attr
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		switch c.Type(htmlLang) {
		case "attribute_name":
			a.name = strings.ToLower(c.Text(src))
		case "attribute_value":
			a.value = html.UnescapeString(c.Text(src))
		case "quoted_attribute_value":
			for j := 0; j < c.ChildCount(); j++ {
				if c.Child(j).Type(htmlLang) == "attribute_value" {
					a.value = html.UnescapeString(c.Child(j).Text(src))
				}
			}
		}
	}
	return a
}
