// Package renderhtml writes a Resolved EmailDoc to the pixel-targeted HTML
// part: theme tokens become inline styles, entities decoded by gosx at
// compile time are re-escaped minimally, and attribute order follows
// source order (spec section 7.2, 7.4).
package renderhtml

import (
	"strconv"
	"strings"

	"m31labs.dev/gsxmail/doc"
)

// Write renders resolved to a full HTML document string using theme.
func Write(resolved *doc.Resolved, theme Theme) string {
	var b strings.Builder
	writeHead(&b, theme, resolved.Shell)
	writeBodyOpen(&b, theme, resolved.Shell)
	for _, block := range resolved.Blocks {
		writeBlock(&b, theme, block)
	}
	writeBodyClose(&b)
	return b.String()
}

func writeHead(b *strings.Builder, theme Theme, shell doc.ResolvedShell) {
	b.WriteString("<!DOCTYPE html>\n<html lang=\"")
	b.WriteString(escapeAttr(shell.Lang))
	b.WriteString("\">\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	if theme.ColorScheme != "" {
		b.WriteString("<meta name=\"color-scheme\" content=\"")
		b.WriteString(escapeAttr(theme.ColorScheme))
		b.WriteString("\">\n<meta name=\"supported-color-schemes\" content=\"")
		b.WriteString(escapeAttr(theme.ColorScheme))
		b.WriteString("\">\n")
	}
	b.WriteString("<title>")
	b.WriteString(escapeText(shell.Title))
	b.WriteString("</title>\n</head>\n")
}

func writeBodyOpen(b *strings.Builder, theme Theme, shell doc.ResolvedShell) {
	width := strconv.Itoa(theme.CardWidth)

	b.WriteString(`<body style="margin:0; padding:0; background-color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%; background-color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`; margin:0; padding:0;">
<tr>
<td align="center" style="padding:32px 16px;">
<table role="presentation" width="`)
	b.WriteString(width)
	b.WriteString(`" cellpadding="0" cellspacing="0" border="0" style="width:`)
	b.WriteString(width)
	b.WriteString(`px; max-width:`)
	b.WriteString(width)
	b.WriteString(`px; background-color:`)
	b.WriteString(theme.ColorCard)
	b.WriteString(`; border:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; border-radius:4px;">
<tr>
<td style="padding:24px 32px 20px 32px; border-bottom:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td width="44" valign="middle" style="padding-right:12px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="border:1px solid `)
	b.WriteString(theme.ColorAccent)
	b.WriteString(`; border-radius:2px;"><tr><td style="width:40px; height:40px; text-align:center; vertical-align:middle; color:`)
	b.WriteString(theme.ColorAccent)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontMono)
	b.WriteString(`; font-size:13px; font-weight:700;">`)
	b.WriteString(escapeText(shell.ShortCode))
	b.WriteString(`</td></tr></table>
</td>
<td valign="middle">
<div style="color:`)
	b.WriteString(theme.ColorInk)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:19px; font-weight:800; letter-spacing:-0.02em; text-transform:uppercase;">`)
	b.WriteString(escapeText(shell.Wordmark))
	b.WriteString(`</div>
<div style="color:`)
	b.WriteString(theme.ColorMuted)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontMono)
	b.WriteString(`; font-size:11px; letter-spacing:0.08em; text-transform:uppercase; margin-top:3px;">`)
	b.WriteString(escapeText(shell.Tagline))
	b.WriteString(`</div>
</td>
</tr>
</table>
</td>
</tr>
`)
}

func writeBodyClose(b *strings.Builder) {
	b.WriteString(`</table>
</td>
</tr>
</table>
</body>
</html>
`)
}

func writeBlock(b *strings.Builder, theme Theme, block doc.ResolvedBlock) {
	switch v := block.(type) {
	case doc.ResolvedSignal:
		writeSignal(b, theme, v)
	case doc.ResolvedHeadline:
		writeHeadline(b, theme, v)
	case doc.ResolvedPanel:
		writePanel(b, theme, v)
	case doc.ResolvedCTA:
		writeCTA(b, theme, v)
	case doc.ResolvedPickList:
		writePickList(b, theme, v)
	case doc.ResolvedFooter:
		writeFooter(b, theme, v)
	case doc.ResolvedNote:
		writeNote(b, theme, v)
	case doc.ResolvedDivider:
		writeDivider(b, theme, v)
	case doc.ResolvedStatTable:
		writeStatTable(b, theme, v)
	case doc.ResolvedCustom:
		writeCustomNode(b, v.Root)
	}
}

func writeSignal(b *strings.Builder, theme Theme, s doc.ResolvedSignal) {
	b.WriteString(`<tr>
<td style="padding:20px 32px 0 32px;">
<div style="color:`)
	b.WriteString(theme.ColorAccent)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontMono)
	b.WriteString(`; font-size:12px; letter-spacing:0.08em; text-transform:uppercase;">` + "●" + ` `)
	b.WriteString(escapeText(s.Text))
	b.WriteString(`</div>
</td>
</tr>
`)
}

func writeHeadline(b *strings.Builder, theme Theme, h doc.ResolvedHeadline) {
	b.WriteString(`<tr>
<td style="padding:14px 32px 0 32px;">
<div style="color:`)
	b.WriteString(theme.ColorInk)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:32px; font-weight:800; letter-spacing:-0.03em; text-transform:uppercase; line-height:1.08;">`)
	b.WriteString(escapeText(h.Title))
	b.WriteString(`</div>
<div style="color:`)
	b.WriteString(theme.ColorBody)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:15px; line-height:1.6; margin-top:12px;">`)
	b.WriteString(escapeText(h.Lede))
	b.WriteString(`</div>
</td>
</tr>
`)
}

func writePanel(b *strings.Builder, theme Theme, p doc.ResolvedPanel) {
	b.WriteString(`<tr>
<td style="padding:24px 32px 0 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%; border:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; border-radius:4px; background-color:`)
	b.WriteString(theme.ColorPanel)
	b.WriteString(`;">
`)
	for i, row := range p.Rows {
		b.WriteString(`<tr>
<td style="padding:16px 20px;`)
		if i < len(p.Rows)-1 {
			b.WriteString(` border-bottom:1px solid `)
			b.WriteString(theme.ColorBorder)
			b.WriteString(`;`)
		}
		b.WriteString(`">
<span style="display:inline-block; width:88px; color:`)
		b.WriteString(theme.ColorMuted)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontMono)
		b.WriteString(`; font-size:11px; letter-spacing:0.06em; text-transform:uppercase; vertical-align:top;">`)
		b.WriteString(escapeText(row.Label))
		b.WriteString(`</span>
<span style="color:`)
		b.WriteString(theme.ColorInk)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontSans)
		b.WriteString(`; font-size:14px;">`)
		b.WriteString(escapeText(row.Value))
		b.WriteString(`</span>
</td>
</tr>
`)
	}
	b.WriteString(`</table>
</td>
</tr>
`)
}

func writeCTA(b *strings.Builder, theme Theme, c doc.ResolvedCTA) {
	b.WriteString(`<tr>
<td align="center" style="padding:28px 32px 0 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td style="border-radius:2px; background-color:`)
	b.WriteString(theme.ColorAccent)
	b.WriteString(`;">
`)
	faceStyle := "display:inline-block; padding:14px 30px; color:" + theme.ColorGround +
		"; font-family:" + theme.FontSans +
		"; font-size:14px; font-weight:800; letter-spacing:0.04em; text-transform:uppercase; text-decoration:none; border-radius:2px;"
	if hasSafeHrefScheme(c.Href) {
		b.WriteString(`<a href="`)
		b.WriteString(escapeAttr(c.Href))
		b.WriteString(`" style="`)
		b.WriteString(faceStyle)
		b.WriteString(`">`)
		b.WriteString(escapeText(c.Label))
		b.WriteString(`</a>
`)
	} else {
		b.WriteString(`<span style="`)
		b.WriteString(faceStyle)
		b.WriteString(`">`)
		b.WriteString(escapeText(c.Label))
		b.WriteString(`</span>
`)
	}
	b.WriteString(`</td>
</tr>
</table>
</td>
</tr>
`)
}

func writePickList(b *strings.Builder, theme Theme, p doc.ResolvedPickList) {
	b.WriteString(`<tr>
<td style="padding:28px 32px 0 32px;">
`)
	if p.Title != "" {
		b.WriteString(`<div style="color:`)
		b.WriteString(theme.ColorMuted)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontMono)
		b.WriteString(`; font-size:11px; letter-spacing:0.08em; text-transform:uppercase; margin-bottom:12px;">`)
		b.WriteString(escapeText(p.Title))
		b.WriteString(`</div>
`)
	}
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
`)
	for i, item := range p.Items {
		b.WriteString(`<tr><td style="padding:5px 0; color:`)
		b.WriteString(theme.ColorBody)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontSans)
		b.WriteString(`; font-size:14px;"><span style="color:`)
		b.WriteString(theme.ColorAccent)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontMono)
		b.WriteString(`; font-weight:700;">`)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(`.</span>&nbsp; `)
		b.WriteString(escapeText(item))
		b.WriteString(`</td></tr>
`)
	}
	b.WriteString(`</table>
</td>
</tr>
`)
}

func writeNote(b *strings.Builder, theme Theme, n doc.ResolvedNote) {
	b.WriteString(`<tr>
<td style="padding:24px 32px 0 32px;">
<div style="color:`)
	b.WriteString(theme.ColorBody)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:14px; line-height:1.6;">`)
	b.WriteString(escapeText(n.Text))
	b.WriteString(`</div>
</td>
</tr>
`)
}

func writeDivider(b *strings.Builder, theme Theme, _ doc.ResolvedDivider) {
	b.WriteString(`<tr>
<td style="padding:20px 32px 0 32px;">
<div style="border-top:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`;"></div>
</td>
</tr>
`)
}

// writeStatTable ports emailkit's StatTable.appendHTML (blocks.go), driven
// by Theme's tokens instead of package-level constants. MarkRow (1-based;
// 0 means no row is marked) selects the accent-colored row, matching
// emailkit's own MarkRow semantics (design spec section 6.5).
func writeStatTable(b *strings.Builder, theme Theme, s doc.ResolvedStatTable) {
	b.WriteString(`<tr>
<td style="padding:24px 32px 0 32px;">
`)
	if s.Title != "" {
		b.WriteString(`<div style="color:`)
		b.WriteString(theme.ColorMuted)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontMono)
		b.WriteString(`; font-size:11px; letter-spacing:0.08em; text-transform:uppercase; margin-bottom:12px;">`)
		b.WriteString(escapeText(s.Title))
		b.WriteString(`</div>
`)
	}
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%; border:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; border-radius:4px;">
`)
	if len(s.Header) > 0 {
		b.WriteString("<tr>\n")
		for _, cell := range s.Header {
			b.WriteString(`<td style="padding:10px 14px; border-bottom:1px solid `)
			b.WriteString(theme.ColorBorder)
			b.WriteString(`; color:`)
			b.WriteString(theme.ColorMuted)
			b.WriteString(`; font-family:`)
			b.WriteString(theme.FontMono)
			b.WriteString(`; font-size:11px; letter-spacing:0.06em; text-transform:uppercase;">`)
			b.WriteString(escapeText(cell))
			b.WriteString("</td>\n")
		}
		b.WriteString("</tr>\n")
	}
	for i, row := range s.Rows {
		rowColor := theme.ColorBody
		background := ""
		if i+1 == s.MarkRow {
			rowColor = theme.ColorAccent
			background = ` background-color:` + theme.ColorPanel + `;`
		}
		border := ""
		if i < len(s.Rows)-1 {
			border = ` border-bottom:1px solid ` + theme.ColorBorder + `;`
		}
		b.WriteString(`<tr style="`)
		b.WriteString(background)
		b.WriteString("\">\n")
		for _, cell := range row.Cells {
			b.WriteString(`<td style="padding:10px 14px;`)
			b.WriteString(border)
			b.WriteString(` color:`)
			b.WriteString(rowColor)
			b.WriteString(`; font-family:`)
			b.WriteString(theme.FontSans)
			b.WriteString(`; font-size:14px;">`)
			b.WriteString(escapeText(cell))
			b.WriteString("</td>\n")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString(`</table>
</td>
</tr>
`)
}

// voidElements are the allowlisted raw elements with no closing tag.
var voidElements = map[string]bool{"img": true, "br": true, "hr": true}

// writeCustomNode writes one Custom subtree node (design spec section 7.2):
// a raw element with its literal (already lint-checked) attributes, or a
// text run. It carries the subtree through unmodified; gsxmail applies no
// theme tokens to a Custom element, since Custom exists precisely for
// markup the stdlib does not style on the author's behalf.
func writeCustomNode(b *strings.Builder, n doc.ResolvedCustomNode) {
	if n.IsText {
		b.WriteString(escapeText(n.Text))
		return
	}
	b.WriteByte('<')
	b.WriteString(n.Tag)
	for _, a := range n.Attrs {
		b.WriteByte(' ')
		b.WriteString(a.Name)
		b.WriteString(`="`)
		b.WriteString(escapeAttr(a.Value))
		b.WriteByte('"')
	}
	b.WriteByte('>')
	if voidElements[n.Tag] {
		return
	}
	for _, c := range n.Children {
		writeCustomNode(b, c)
	}
	b.WriteString("</")
	b.WriteString(n.Tag)
	b.WriteByte('>')
}

func writeFooter(b *strings.Builder, theme Theme, f doc.ResolvedFooter) {
	b.WriteString(`<tr>
<td style="padding:28px 32px 32px 32px;">
<div style="border-top:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; padding-top:20px; color:`)
	b.WriteString(theme.ColorBody)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:14px;">`)
	b.WriteString(escapeText(f.Signoff))
	b.WriteString(`</div>
<div style="color:`)
	b.WriteString(theme.ColorFaint)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontMono)
	b.WriteString(`; font-size:10px; letter-spacing:0.04em; text-transform:uppercase; margin-top:10px;">`)
	b.WriteString(escapeText(f.Note))
	b.WriteString(`</div>
</td>
</tr>
`)
}
