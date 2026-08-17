package renderhtml

import "strings"

// escapeText escapes s for placement inside element text content: &, <, and
// >, plus U+00A0 (non-breaking space) to the literal entity &nbsp; for diff
// visibility. This is a minimal escaper by
// design: gsxmail owns entity decoding at compile time (gosx decodes
// entities into UTF-8 when it lowers text nodes), so the writer only ever
// needs to re-escape the handful of characters HTML text content cannot
// carry literally.
func escapeText(s string) string {
	return escapeReplacer.Replace(s)
}

// escapeAttr escapes s for placement inside a double-quoted attribute
// value: the same three characters as escapeText, plus the double quote.
func escapeAttr(s string) string {
	return escapeAttrReplacer.Replace(s)
}

var escapeReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	" ", "&nbsp;",
)

var escapeAttrReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\"", "&quot;",
	" ", "&nbsp;",
)
