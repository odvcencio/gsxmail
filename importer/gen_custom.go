package importer

import (
	"fmt"
	"strconv"
	"strings"
)

// qexpr renders s as a gsx {expression} hole holding a Go string literal
// (strconv.Quote correctly escapes quotes, backslashes, and control
// bytes into valid Go source). Every piece of text or attribute value
// Import derives from the source HTML goes through this one function —
// gsx's own {expression} attribute/child syntax accepts a bare Go string
// literal (confirmed against testdata/allblocks/allblocks.gsx's own
// `<img src={props.ImgSrc} alt={props.ImgAlt} />` and
// `<email.Item>{props.PickItem}</email.Item>` shapes), so this sidesteps
// ever having to know gsx's own raw-text-child escaping rules, which are
// not part of this package's own contract to track.
func qexpr(s string) string {
	return "{" + strconv.Quote(s) + "}"
}

// indent prefixes every non-empty line of s with four spaces, used when
// nesting one block's own gsx fragment inside another (StatTable's
// <Each>, PickList's <email.Item> list, Columns' children).
func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l == "" {
			continue
		}
		lines[i] = "    " + l
	}
	return strings.Join(lines, "\n")
}

// safeScheme reports whether scheme is one gsxmail's own EM110 accepts:
// https, http, or mailto.
func safeScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "https", "http", "mailto":
		return true
	}
	return false
}

// placeholderHref is the literal href Import substitutes for a source
// link whose own scheme EM110 would reject ("#", a relative path,
// "javascript:...", or an empty href) — proof-bar item (c) requires the
// generated .gsx to load cleanly, and EM110 is error-severity, so a
// scheme it rejects cannot survive into the emitted source unchanged.
const placeholderHref = "https://example.com/imported-link"

// sanitizeHref returns href unchanged when its scheme is https/http/
// mailto, or placeholderHref with a report note otherwise.
func sanitizeHref(href string) (safe, note string) {
	href = strings.TrimSpace(href)
	scheme := ""
	if i := strings.Index(href, ":"); i > 0 {
		scheme = href[:i]
	}
	if href != "" && safeScheme(scheme) {
		return href, ""
	}
	return placeholderHref, fmt.Sprintf(
		"the source link %q could not be resolved to a safe https/http/mailto href; replaced it with a placeholder",
		href)
}

// placeholderImgSrc is the literal src Import substitutes for a source
// image whose own src is not an absolute https URL — EM111 requires one.
const placeholderImgSrc = "https://example.com/imported-asset.png"

func sanitizeImgSrc(src string) (safe, note string) {
	src = strings.TrimSpace(src)
	if strings.HasPrefix(src, "https://") {
		return src, ""
	}
	return placeholderImgSrc, fmt.Sprintf(
		"the source image %q is not an absolute https URL; replaced it with a placeholder — point it at the real asset",
		src)
}

// customElementAllowlist mirrors lint's own elementAllowlist (EM003):
// only these raw tags may appear inside a Custom fallback subtree.
var customElementAllowlist = map[string]bool{
	"table": true, "tr": true, "td": true, "th": true,
	"div": true, "span": true, "p": true, "a": true, "img": true,
	"br": true, "hr": true,
	"h1": true, "h2": true, "h3": true, "h4": true,
	"strong": true, "em": true,
	"ul": true, "ol": true, "li": true,
}

// customTagRemap maps a common non-allowlisted tag onto its closest
// allowlisted equivalent, so ordinary legacy markup (table-soup <font>,
// <b>, <i>, <center>, HTML5 sectioning elements) survives inside a
// Custom block instead of being dropped outright.
var customTagRemap = map[string]string{
	"font": "span", "b": "strong", "i": "em", "center": "div",
	"section": "div", "article": "div", "header": "div", "footer": "div",
	"nav": "div", "aside": "div", "main": "div", "small": "span",
	"h5": "h4", "h6": "h4", "pre": "p", "blockquote": "p", "code": "span",
	"label": "span",
}

// dropEntirely names tags whose whole subtree carries no email content
// (document/head/metadata plumbing, or something gsxmail already
// forbids outright): they, and their children, are skipped rather than
// unwrapped.
var dropEntirely = map[string]bool{
	"html": true, "head": true, "body": true, "meta": true, "title": true,
	"style": true, "script": true, "link": true, "noscript": true,
	"xml": true, "colgroup": true, "col": true, "svg": true, "video": true,
	"audio": true, "iframe": true, "!doctype": true, "input": true, "form": true,
}

// writeCustomFallback serializes row — the whole, unmapped card row — as
// a sanitized raw-HTML fragment (task instructions, item 3: "every node
// the mapper could not confidently place lands in an email.Custom
// escape-hatch block ... preserving its subtree"). It always wraps the
// result in a single-cell table so a stray top-level text run or bare
// inline element still nests inside an EM003-allowed structural element,
// matching gsxmail's own row-per-block shape elsewhere in the card.
func writeCustomFallback(row *node) string {
	var b strings.Builder
	b.WriteString("<table style=\"width:100%;\">\n<tr>\n<td>\n")
	for _, c := range row.elements() {
		writeCustomNode(&b, c)
	}
	// A row with no element children at all (bare text only, unusual but
	// not impossible from a malformed source) falls back to its own
	// flattened text.
	if len(row.elements()) == 0 {
		if t := strings.TrimSpace(row.innerText()); t != "" {
			b.WriteString(qexpr(t))
			b.WriteString("\n")
		}
	}
	b.WriteString("</td>\n</tr>\n</table>\n")
	return b.String()
}

// writeWholeBodyCustom is writeCustomFallback's whole-document fallback:
// findCard found no plausible card table at all, so Import preserves
// body's own element children, sanitized, as a sequence of Custom
// elements directly under Shell rather than as one row's worth of
// content.
func writeWholeBodyCustom(body *node) string {
	var b strings.Builder
	b.WriteString("<table style=\"width:100%;\">\n<tr>\n<td>\n")
	for _, c := range body.elements() {
		writeCustomNode(&b, c)
	}
	b.WriteString("</td>\n</tr>\n</table>\n")
	return b.String()
}

// writeCustomNode serializes one node (and its subtree) as sanitized raw
// gsx markup: a non-allowlisted tag is remapped (customTagRemap) or
// unwrapped to its children (dropEntirely, or an unknown tag with
// neither a remap nor a drop rule), "class" and event-handler attributes
// are stripped (EM104/EM004), "href" is scheme-sanitized (EM110), "img"
// keeps only an https src and a non-empty alt (EM111/EM112), and "style"
// is filtered to properties the embedded caniemail matrix never marks
// unsupported (EM101) — see style.go's sanitizeStyle.
func writeCustomNode(b *strings.Builder, n *node) {
	if n.isComment || n.errorish {
		return // gotreesitter's own error-recovery nodes carry no reliable shape to preserve
	}
	if n.isText {
		if t := strings.TrimSpace(n.text); t != "" {
			b.WriteString(qexpr(collapseSpace(n.text)))
		}
		return
	}
	tag := n.tag
	if dropEntirely[tag] {
		return
	}
	if remap, ok := customTagRemap[tag]; ok {
		tag = remap
	}
	if !customElementAllowlist[tag] {
		// An unrecognized tag with no remap rule: unwrap to its children
		// rather than dropping real content outright.
		for _, c := range n.elements() {
			writeCustomNode(b, c)
		}
		for _, c := range n.children {
			if c.isText {
				writeCustomNode(b, &c)
			}
		}
		return
	}

	b.WriteByte('<')
	b.WriteString(tag)
	for _, a := range n.attrs {
		writeCustomAttr(b, tag, a)
	}
	if tag == "img" {
		src, _ := n.attr("src")
		src, _ = sanitizeImgSrc(src)
		alt, hasAlt := n.attr("alt")
		if !hasAlt || strings.TrimSpace(alt) == "" {
			alt = "Imported image"
		}
		b.WriteString(" src=" + qexpr(src) + " alt=" + qexpr(alt))
		b.WriteString(" />")
		return
	}
	b.WriteByte('>')
	if tag == "br" || tag == "hr" {
		return
	}
	for i := range n.children {
		writeCustomNode(b, &n.children[i])
	}
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteByte('>')
}

// writeCustomAttr writes one sanitized attribute for a Custom tag.
// "class" (EM104) and any "on*" event handler (EM004) are dropped
// outright; "src"/"alt" on <img> are handled by the img special case in
// writeCustomNode instead (both always need a value, even when the
// source carried neither); everything else passes through as a quoted
// {expression} hole, href and style sanitized first.
func writeCustomAttr(b *strings.Builder, tag string, a attr) {
	name := strings.ToLower(a.name)
	if name == "class" || strings.HasPrefix(name, "on") {
		return
	}
	if tag == "img" && (name == "src" || name == "alt") {
		return
	}
	value := a.value
	switch name {
	case "href":
		value, _ = sanitizeHref(value)
	case "style":
		value, _ = sanitizeStyle(value)
		if value == "" {
			return
		}
	}
	b.WriteByte(' ')
	b.WriteString(name)
	b.WriteByte('=')
	b.WriteString(qexpr(value))
}
