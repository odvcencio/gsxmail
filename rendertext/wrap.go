package rendertext

import (
	"strings"
	"unicode/utf8"
)

// WrapWidth is the text-part wrap column. A word joins the current line
// only while the resulting line stays strictly under this column; a
// candidate that would reach or pass it starts a new line instead (spec
// section 9, the emailkit rules carried over verbatim).
const WrapWidth = 72

// wrapText greedily wraps s into lines of at most width-1 display columns.
// Width is measured in runes, not bytes, so a multi-byte character such as
// "·" or "—" counts as the one column it occupies. A single word at or past
// width is never broken (a long URL or token still occupies one line — the
// caller decides whether width even applies; table rows and URLs never
// wrap). Multiple whitespace runs, including embedded newlines, collapse to
// single spaces. An empty or all-whitespace s returns nil. Ported from
// gridiron's internal/emailkit/text.go.
func wrapText(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, 4)
	current := words[0]
	currentWidth := utf8.RuneCountInString(current)
	for _, word := range words[1:] {
		wordWidth := utf8.RuneCountInString(word)
		candidateWidth := currentWidth + 1 + wordWidth
		if candidateWidth < width {
			current = current + " " + word
			currentWidth = candidateWidth
			continue
		}
		lines = append(lines, current)
		current = word
		currentWidth = wordWidth
	}
	lines = append(lines, current)
	return lines
}

// padLabel left-justifies label to width display columns with trailing
// spaces, measuring width in runes rather than bytes.
func padLabel(label string, width int) string {
	n := utf8.RuneCountInString(label)
	if n >= width {
		return label
	}
	return label + strings.Repeat(" ", width-n)
}

// panelValueColumn returns the column at which every panel row's value (and
// every continuation line of a wrapped value) begins: a 2-space indent, the
// longest label, then 4 spaces — aligned to the longest label plus 4
// spaces, emailkit's own convention. Label width is measured in
// runes.
func panelValueColumn(labels []string) int {
	longest := 0
	for _, label := range labels {
		if n := utf8.RuneCountInString(label); n > longest {
			longest = n
		}
	}
	return 2 + longest + 4
}

// columnWidths computes the display width of each column across a header
// row and every data row, so a text table's columns line up without
// wrapping any cell (table rows never wrap). Width is
// measured in runes. The column count is sized from whichever is wider,
// the header or the widest data row, so a table with no header still
// sizes every row cell's own column correctly. Ported from gridiron's
// internal/emailkit/text.go.
func columnWidths(header []string, rows [][]string) []int {
	numCols := len(header)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	widths := make([]int, numCols)
	for i, cell := range header {
		widths[i] = utf8.RuneCountInString(cell)
	}
	for _, row := range rows {
		for i, cell := range row {
			if n := utf8.RuneCountInString(cell); n > widths[i] {
				widths[i] = n
			}
		}
	}
	return widths
}

// joinRow renders one table row with each cell left-justified to its
// column width and a 2-space gutter between columns. Ported from gridiron's
// internal/emailkit/text.go.
func joinRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		if i < len(widths)-1 {
			parts[i] = padLabel(cell, widths[i])
			continue
		}
		parts[i] = cell // last column never trails with padding
	}
	return strings.Join(parts, "  ")
}

// dashRow renders the underline row beneath a text table's header: one
// run of dashes per column, sized to that column's width. Ported from
// gridiron's internal/emailkit/text.go.
func dashRow(widths []int) string {
	cells := make([]string, len(widths))
	for i, width := range widths {
		cells[i] = strings.Repeat("-", width)
	}
	return strings.Join(cells, "  ")
}
