package rendertext

import (
	"strings"

	"m31labs.dev/gsxmail/doc"
)

// customBlockTags are the Custom-subtree elements (design spec section 9)
// that start a new paragraph in the derived text. table and its own
// tr/td/th are handled by deriveTableText instead (a table's rows never
// wrap); a, img, br, and hr each derive their own fixed shape.
var customBlockTags = map[string]bool{
	"div": true, "p": true, "li": true, "ul": true, "ol": true,
	"h1": true, "h2": true, "h3": true, "h4": true,
}

// writeCustom derives a Custom subtree's text form structurally (design
// spec section 9): block elements start a new paragraph; <a> renders
// "label (url)" (or just the url when the label matches it); <img> renders
// its alt text in brackets; a <table> renders as two-space-guttered
// columns with a dashed underline under a <th> header row, if any; any
// element's own "text" attribute overrides its derived content outright.
// Every paragraph wraps at 72 columns except a table's own rows, which
// never wrap — the same rule StatTable's own text twin follows.
func writeCustom(b *strings.Builder, root doc.ResolvedCustomNode) {
	b.WriteString(deriveCustomText(root))
}

func deriveCustomText(n doc.ResolvedCustomNode) string {
	if n.IsText {
		return n.Text
	}
	if override, ok := attrValue(n.Attrs, "text"); ok {
		return override
	}
	switch n.Tag {
	case "table":
		return deriveTableText(n)
	case "a":
		return deriveLinkText(n)
	case "img":
		alt, _ := attrValue(n.Attrs, "alt")
		return "[" + alt + "]"
	case "br":
		return ""
	case "hr":
		return strings.Repeat("-", WrapWidth-1)
	}

	// Every other allowlisted tag (div, p, li, ul, ol, h1-h4, span,
	// strong, em, td/th/tr reached outside a table) walks its children:
	// a block-tagged child starts its own paragraph; anything else joins
	// the surrounding inline run.
	var paragraphs []string
	var inline strings.Builder
	flush := func() {
		if inline.Len() == 0 {
			return
		}
		paragraphs = append(paragraphs, inline.String())
		inline.Reset()
	}
	for _, c := range n.Children {
		if !c.IsText && customBlockTags[c.Tag] {
			flush()
			if t := deriveCustomText(c); t != "" {
				paragraphs = append(paragraphs, t)
			}
			continue
		}
		inline.WriteString(deriveCustomText(c))
	}
	flush()
	joined := strings.Join(paragraphs, "\n\n")
	if customBlockTags[n.Tag] && !strings.Contains(joined, "\n") {
		// A lone inline run inside a block element still wraps at 72
		// columns; a result already joined from several block children
		// (each wrapped by its own recursive call) is left alone.
		joined = strings.Join(wrapText(joined, WrapWidth), "\n")
	}
	return joined
}

// deriveTableText renders a Custom <table>'s rows as two-space-guttered
// columns (spec section 9; the same shape StatTable's own text twin
// uses). A first row made entirely of <th> cells becomes the header, with
// a dashed underline; every other row's cells never wrap.
func deriveTableText(table doc.ResolvedCustomNode) string {
	var header []string
	var rows [][]string
	hasHeader := false
	rowIndex := 0
	for _, tr := range table.Children {
		if tr.IsText || tr.Tag != "tr" {
			continue
		}
		var cells []string
		allTh := true
		for _, cell := range tr.Children {
			if cell.IsText || (cell.Tag != "td" && cell.Tag != "th") {
				continue
			}
			if cell.Tag != "th" {
				allTh = false
			}
			cells = append(cells, cellText(cell))
		}
		if rowIndex == 0 && allTh && len(cells) > 0 {
			header = cells
			hasHeader = true
		} else {
			rows = append(rows, cells)
		}
		rowIndex++
	}

	widths := columnWidths(header, rows)
	var lines []string
	if hasHeader {
		lines = append(lines, "  "+joinRow(header, widths))
		lines = append(lines, "  "+dashRow(widths))
	}
	for _, row := range rows {
		lines = append(lines, "  "+joinRow(row, widths))
	}
	return strings.Join(lines, "\n")
}

// cellText derives a table cell's own text without wrapping (table rows
// never wrap) and collapses internal whitespace to single spaces, so a
// cell's content always renders as one line.
func cellText(cell doc.ResolvedCustomNode) string {
	if override, ok := attrValue(cell.Attrs, "text"); ok {
		return override
	}
	var b strings.Builder
	for _, c := range cell.Children {
		b.WriteString(deriveCustomText(c))
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// deriveLinkText renders <a> as "label (url)" (design spec section 9),
// unless the label is empty or equal to the url, in which case the url
// alone is enough.
func deriveLinkText(a doc.ResolvedCustomNode) string {
	var b strings.Builder
	for _, c := range a.Children {
		b.WriteString(deriveCustomText(c))
	}
	label := strings.TrimSpace(b.String())
	href, hasHref := attrValue(a.Attrs, "href")
	switch {
	case !hasHref || href == "":
		return label
	case label == "" || label == href:
		return href
	default:
		return label + " (" + href + ")"
	}
}

func attrValue(attrs []doc.ResolvedCustomAttr, name string) (string, bool) {
	for _, a := range attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}
