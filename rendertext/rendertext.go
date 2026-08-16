// Package rendertext writes a Resolved EmailDoc to its 72-column plain-text
// twin, using the emailkit wrap/column rules (spec section 9): every
// stdlib block derives its text form from the same resolved values the
// HTML writer sees, so the two parts cannot drift by construction.
package rendertext

import (
	"net/url"
	"strconv"
	"strings"

	"m31labs.dev/gsxmail/doc"
)

// Write renders resolved to the plain-text part.
func Write(resolved *doc.Resolved) string {
	var b strings.Builder
	b.WriteString(resolved.Shell.Wordmark)
	b.WriteString(" // ")
	b.WriteString(resolved.Shell.Tagline)
	for _, block := range resolved.Blocks {
		// A Divider carries its blank line as the ambient separator
		// between its neighbors, not as its own extra pair of separators
		// (ported from gridiron's internal/emailkit finding nit 1): skip
		// its own leading "\n\n" entirely, since writeBlock already emits
		// nothing for it. Without this, the "\n\n" written here plus the
		// next block's own leading "\n\n" would stack into three blank
		// lines instead of the spec's one.
		//
		// Spacer (WP5.3; pixel dossier section 4.8) skips the same way and
		// for the same arithmetic reason: text has no concept of a
		// variable-height gap, so every Spacer height folds to the one
		// blank line the ambient separator already provides — a 24px
		// Spacer and a 48px Spacer render identical text. This is also
		// why Spacer's own writeBlock case below is empty, the same shape
		// Divider's is.
		switch block.(type) {
		case doc.ResolvedDivider, doc.ResolvedSpacer:
			continue
		}
		b.WriteString("\n\n")
		writeBlock(&b, block)
	}
	return b.String()
}

func writeBlock(b *strings.Builder, block doc.ResolvedBlock) {
	switch v := block.(type) {
	case doc.ResolvedSignal:
		writeSignal(b, v)
	case doc.ResolvedHeadline:
		writeHeadline(b, v)
	case doc.ResolvedPanel:
		writePanel(b, v)
	case doc.ResolvedCTA:
		writeCTA(b, v)
	case doc.ResolvedPickList:
		writePickList(b, v)
	case doc.ResolvedFooter:
		writeFooter(b, v)
	case doc.ResolvedNote:
		writeNote(b, v)
	case doc.ResolvedDivider:
		// No output of its own; see Write's Divider special case above.
	case doc.ResolvedStatTable:
		writeStatTable(b, v)
	case doc.ResolvedCustom:
		writeCustom(b, v.Root)
	case doc.ResolvedButton:
		writeButton(b, v)
	case doc.ResolvedColumns:
		writeColumns(b, v)
	case doc.ResolvedHero:
		writeHero(b, v)
	case doc.ResolvedSpacer:
		// No output of its own; Write's own loop skips Spacer the same way
		// it skips Divider, so the ambient single "\n\n" between its
		// neighbors is Spacer's entire text-twin derivation ("Spacer
		// renders one blank line" — pixel dossier section 4.8's own
		// framing, WP5.3). See Write's own doc comment for the arithmetic:
		// without the skip, Spacer's own leading "\n\n" plus the next
		// block's leading "\n\n" would stack into three blank lines
		// instead of one.
	case doc.ResolvedBadge:
		writeBadge(b, v)
	}
}

func writeSignal(b *strings.Builder, s doc.ResolvedSignal) {
	b.WriteString("* ")
	b.WriteString(s.Text)
}

func writeHeadline(b *strings.Builder, h doc.ResolvedHeadline) {
	b.WriteString(h.Title)
	if h.Lede == "" {
		return
	}
	b.WriteString("\n\n")
	b.WriteString(strings.Join(wrapText(h.Lede, WrapWidth), "\n"))
}

func writePanel(b *strings.Builder, p doc.ResolvedPanel) {
	if len(p.Rows) == 0 {
		return
	}
	labels := make([]string, len(p.Rows))
	for i, row := range p.Rows {
		labels[i] = row.Label
	}
	col := panelValueColumn(labels)
	maxLabel := col - 2 - 4
	valueWidth := WrapWidth - col
	lines := make([]string, 0, len(p.Rows))
	for _, row := range p.Rows {
		wrapped := wrapText(row.Value, valueWidth)
		if len(wrapped) == 0 {
			wrapped = []string{""}
		}
		lines = append(lines, "  "+padLabel(row.Label, maxLabel)+"    "+wrapped[0])
		indent := strings.Repeat(" ", col)
		for _, cont := range wrapped[1:] {
			lines = append(lines, indent+cont)
		}
	}
	b.WriteString(strings.Join(lines, "\n"))
}

func writeCTA(b *strings.Builder, c doc.ResolvedCTA) {
	// strings.TrimSuffix removes exactly the literal " →" once; TrimRight
	// would treat " →" as a cutset of individual characters and over-trim
	// (a gridiron/emailkit finding carried over here).
	label := strings.TrimSuffix(c.Label, " →")
	b.WriteString("  -> ")
	b.WriteString(label)
	if hasSafeHrefScheme(c.Href) {
		b.WriteString(": ")
		b.WriteString(c.Href)
	}
}

// writeButton derives email.Button's text form identically for every
// variant (design note: a button's variant is a visual/clickable-area
// choice only; the text twin has no visual axis to encode it in). Every
// variant renders "-> LABEL: URL" — the same shape email.CTA's own
// writeCTA below already used, since Button variant="primary" is CTA's
// alias.
func writeButton(b *strings.Builder, v doc.ResolvedButton) {
	label := strings.TrimSuffix(v.Label, " →")
	b.WriteString("  -> ")
	b.WriteString(label)
	if hasSafeHrefScheme(v.Href) {
		b.WriteString(": ")
		b.WriteString(v.Href)
	}
}

// writeColumns derives email.Columns' text form (pixel dossier section
// 4.9): columns stack in source order as sequential text, each its own
// block, separated by a blank line — never side-by-side ASCII columns,
// since a fixed 72-column width leaves no room for a genuine multi-column
// layout.
func writeColumns(b *strings.Builder, v doc.ResolvedColumns) {
	var blocks []string
	for _, col := range v.Columns {
		if t := columnText(col); t != "" {
			blocks = append(blocks, t)
		}
	}
	b.WriteString(strings.Join(blocks, "\n\n"))
}

// columnText derives one Column's own text: its image's alt text in
// brackets (when set), its title, then its wrapped body text — the same
// "[alt]" convention a Custom <img> already uses (rendertext/custom.go).
func columnText(col doc.ResolvedColumn) string {
	var parts []string
	if col.ImgAlt != "" {
		parts = append(parts, "["+col.ImgAlt+"]")
	}
	if col.Title != "" {
		parts = append(parts, col.Title)
	}
	if col.Text != "" {
		parts = append(parts, strings.Join(wrapText(col.Text, WrapWidth), "\n"))
	}
	return strings.Join(parts, "\n")
}

// writeHero derives email.Hero's text form: its alt text in brackets (the
// same "[alt]" convention a Custom <img> already uses), or nothing when
// Alt is empty. Lint's checkHero reuses EM112 to require Alt non-empty at
// Load time, so the empty case is unreachable through Load; the writer
// stays defensive about it rather than assuming that guarantee holds for
// every possible caller of doc.Resolve directly.
func writeHero(b *strings.Builder, v doc.ResolvedHero) {
	if v.Alt == "" {
		return
	}
	b.WriteString("[")
	b.WriteString(v.Alt)
	b.WriteString("]")
}

// writeBadge derives email.Badge's text form: its label in brackets
// (pixel dossier section 4.11's own worked example states this exact
// shape: "the text twin renders [PAID]"), regardless of tone — tone is a
// color/structural cue with no separate text-safe encoding beyond the
// label itself; the brackets alone already carry "this is a marked
// status," matching the img-alt and StatTable-mark conventions elsewhere
// in the text writer.
func writeBadge(b *strings.Builder, v doc.ResolvedBadge) {
	b.WriteString("[")
	b.WriteString(v.Text)
	b.WriteString("]")
}

func writePickList(b *strings.Builder, p doc.ResolvedPickList) {
	var lines []string
	if p.Title != "" {
		lines = append(lines, p.Title)
	}
	for i, item := range p.Items {
		lines = append(lines, "  "+strconv.Itoa(i+1)+". "+item)
	}
	b.WriteString(strings.Join(lines, "\n"))
}

func writeNote(b *strings.Builder, n doc.ResolvedNote) {
	b.WriteString(strings.Join(wrapText(n.Text, WrapWidth), "\n"))
}

// writeStatTable ports emailkit's StatTable.appendText (text.go/blocks.go):
// an optional title line, an optional header row with a dashed underline,
// then every row two-space-guttered — table rows never wrap. MarkRow
// (1-based; 0 means none) prefixes its row with "* " instead of "  ", so
// the marked row's meaning survives in plain text too (design spec section
// 6.5, "Mark semantics match emailkit's MarkRow").
func writeStatTable(b *strings.Builder, s doc.ResolvedStatTable) {
	var lines []string
	if s.Title != "" {
		lines = append(lines, s.Title)
	}

	rows := make([][]string, len(s.Rows))
	for i, row := range s.Rows {
		rows[i] = row.Cells
	}
	widths := columnWidths(s.Header, rows)
	if len(s.Header) > 0 {
		lines = append(lines, "  "+joinRow(s.Header, widths))
		lines = append(lines, "  "+dashRow(widths))
	}
	for i, row := range rows {
		marker := "  "
		if i+1 == s.MarkRow {
			marker = "* "
		}
		lines = append(lines, marker+joinRow(row, widths))
	}
	b.WriteString(strings.Join(lines, "\n"))
}

func writeFooter(b *strings.Builder, f doc.ResolvedFooter) {
	b.WriteString(f.Signoff)
	b.WriteString("\n")
	b.WriteString(f.Note)
}

// hasSafeHrefScheme mirrors renderhtml's CTA scheme allowlist (https,
// http, mailto — spec section 8, EM110) so the text part's "-> LABEL: URL"
// suffix appears exactly when the HTML part's href does.
func hasSafeHrefScheme(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto":
		return true
	default:
		return false
	}
}
