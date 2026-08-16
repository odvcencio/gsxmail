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
