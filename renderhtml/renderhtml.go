// Package renderhtml writes a Resolved EmailDoc to the pixel-targeted HTML
// part: theme tokens become inline styles, entities decoded by gosx at
// compile time are re-escaped minimally, and attribute order follows
// source order (spec section 7.2, 7.4).
//
// # Output contracts
//
// Write (and WriteWithOptions) emit hardened, bulletproof markup by
// default: role="presentation" on every layout table, an Outlook ghost
// table around the 600px card, doubled width attributes/CSS for the DPI
// fix, and per-component contract details the pixel dossier's section 4
// states (design spec section 15, WP5.1). WriteOptions.Outlook == "off"
// selects parity mode instead: the exact WP1 byte stream, unchanged,
// for a consumer whose own equivalence test pins the old bytes (gsxmail's
// own gridiron invite DOM-parity guard is the example — see
// WriteOptions's doc comment).
package renderhtml

import (
	"strconv"
	"strings"

	"m31labs.dev/gsxmail/doc"
)

// WriteOptions configures Write's output contract (design spec section 15,
// WP5.1; pixel dossier section 4.2's "Shell options" surface).
type WriteOptions struct {
	// Outlook selects the layout technique. "" and "ghost-tables" (the
	// default) emit the hardened contract: an Outlook ghost table around
	// the card, role="presentation" plus border/cellpadding/cellspacing
	// on every layout table, doubled width attributes for the Outlook DPI
	// fix, td-pair Panel rows, an <h1> Headline title, mso-padding-alt on
	// the CTA, a real StatTable data-table contract, and the border-left
	// Note / spacer-technique Divider. "off" emits the WP1 byte stream
	// unchanged — the parity mode a consumer's own byte- or DOM-equivalence
	// test can pin (pixel dossier section 4: "Parity mode ... keeps the
	// WP1 byte stream").
	Outlook string
}

// hardened reports whether opts selects the hardened output contract (the
// default) rather than WP1 parity mode.
func (opts WriteOptions) hardened() bool {
	return opts.Outlook != "off"
}

// Write renders resolved to a full HTML document string using theme, in
// the hardened output contract (WriteOptions{}'s default).
func Write(resolved *doc.Resolved, theme Theme) string {
	return WriteWithOptions(resolved, theme, WriteOptions{})
}

// WriteWithOptions is Write with an explicit WriteOptions (added WP5.1;
// see the package doc and WriteOptions for the two output contracts).
func WriteWithOptions(resolved *doc.Resolved, theme Theme, opts WriteOptions) string {
	hard := opts.hardened()
	var b strings.Builder
	writeHead(&b, theme, resolved.Shell, hard)
	writeBodyOpen(&b, theme, resolved.Shell, hard)
	for _, block := range resolved.Blocks {
		writeBlock(&b, theme, block, hard)
	}
	writeBodyClose(&b, hard)
	return b.String()
}

func writeHead(b *strings.Builder, theme Theme, shell doc.ResolvedShell, hard bool) {
	if !hard {
		writeHeadParity(b, theme, shell)
		return
	}
	b.WriteString("<!DOCTYPE html>\n<html lang=\"")
	b.WriteString(escapeAttr(shell.Lang))
	b.WriteString(`" dir="ltr" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
`)
	if theme.ColorScheme != "" {
		b.WriteString("<meta name=\"color-scheme\" content=\"")
		b.WriteString(escapeAttr(theme.ColorScheme))
		b.WriteString("\">\n<meta name=\"supported-color-schemes\" content=\"")
		b.WriteString(escapeAttr(theme.ColorScheme))
		b.WriteString("\">\n")
	}
	b.WriteString("<title>")
	b.WriteString(escapeText(shell.Title))
	b.WriteString(`</title>
<!--[if mso]>
<noscript>
<xml>
<o:OfficeDocumentSettings>
<o:PixelsPerInch>96</o:PixelsPerInch>
</o:OfficeDocumentSettings>
</xml>
</noscript>
<![endif]-->
<style>
body{margin:0;padding:0;-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%;}
table,td{border-collapse:collapse;mso-table-lspace:0pt;mso-table-rspace:0pt;}
img{border:0;height:auto;line-height:100%;outline:none;text-decoration:none;-ms-interpolation-mode:bicubic;}
#outlook a{padding:0;}
a[x-apple-data-detectors]{color:inherit !important;text-decoration:none !important;}
@media only screen and (max-width:480px){
.gsx-card{width:100% !important;}
.gsx-col{display:block !important;width:100% !important;max-width:100% !important;}
}
</style>
</head>
`)
}

// writeHeadParity is WP1's exact head writer, byte-for-byte (design spec
// section 6.4). It stays untouched so WriteOptions{Outlook:"off"} keeps
// pinning the original bytes.
func writeHeadParity(b *strings.Builder, theme Theme, shell doc.ResolvedShell) {
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

func writeBodyOpen(b *strings.Builder, theme Theme, shell doc.ResolvedShell, hard bool) {
	if !hard {
		writeBodyOpenParity(b, theme, shell)
		return
	}
	width := strconv.Itoa(theme.CardWidth)

	b.WriteString(`<body class="gsx-body" style="margin:0; padding:0; background-color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`;">
<div role="article" aria-roledescription="email" aria-label="`)
	b.WriteString(escapeAttr(shell.Title))
	b.WriteString(`" lang="`)
	b.WriteString(escapeAttr(shell.Lang))
	b.WriteString(`" dir="ltr">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%; background-color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`; margin:0; padding:0;">
<tr>
<td align="center" style="padding:32px 16px;">
<!--[if mso | IE]><table role="presentation" align="center" border="0" cellpadding="0" cellspacing="0" width="`)
	b.WriteString(width)
	b.WriteString(`" style="width:`)
	b.WriteString(width)
	b.WriteString(`px;"><tr><td><![endif]-->
<table role="presentation" width="`)
	b.WriteString(width)
	b.WriteString(`" cellpadding="0" cellspacing="0" border="0" class="gsx-card" style="width:`)
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
<td width="44" valign="middle" style="width:44px; padding-right:12px;">
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

// writeBodyOpenParity is WP1's exact body-open writer, byte-for-byte
// (design spec section 6.4). It stays untouched so WriteOptions{Outlook:
// "off"} keeps pinning the original bytes (the gridiron invite DOM-parity
// guard).
func writeBodyOpenParity(b *strings.Builder, theme Theme, shell doc.ResolvedShell) {
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

func writeBodyClose(b *strings.Builder, hard bool) {
	if !hard {
		b.WriteString(`</table>
</td>
</tr>
</table>
</body>
</html>
`)
		return
	}
	b.WriteString(`</table>
<!--[if mso | IE]></td></tr></table><![endif]-->
</td>
</tr>
</table>
</div>
</body>
</html>
`)
}

func writeBlock(b *strings.Builder, theme Theme, block doc.ResolvedBlock, hard bool) {
	switch v := block.(type) {
	case doc.ResolvedSignal:
		writeSignal(b, theme, v)
	case doc.ResolvedHeadline:
		writeHeadline(b, theme, v, hard)
	case doc.ResolvedPanel:
		writePanel(b, theme, v, hard)
	case doc.ResolvedCTA:
		writeCTA(b, theme, v, hard)
	case doc.ResolvedPickList:
		writePickList(b, theme, v)
	case doc.ResolvedFooter:
		writeFooter(b, theme, v)
	case doc.ResolvedNote:
		writeNote(b, theme, v, hard)
	case doc.ResolvedDivider:
		writeDivider(b, theme, v, hard)
	case doc.ResolvedStatTable:
		writeStatTable(b, theme, v, hard)
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

// writeHeadline writes email.Headline. Hardened mode promotes the title to
// a semantic <h1> with margins zeroed (pixel dossier section 4.3, point
// 2): screen-reader navigation gains a heading at no visual cost. Parity
// mode keeps WP1's <div> title, byte-for-byte.
func writeHeadline(b *strings.Builder, theme Theme, h doc.ResolvedHeadline, hard bool) {
	titleTag := "div"
	titleStyleExtra := ""
	if hard {
		titleTag = "h1"
		titleStyleExtra = "margin:0; "
	}
	b.WriteString(`<tr>
<td style="padding:14px 32px 0 32px;">
<`)
	b.WriteString(titleTag)
	b.WriteString(` style="`)
	b.WriteString(titleStyleExtra)
	b.WriteString(`color:`)
	b.WriteString(theme.ColorInk)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:32px; font-weight:800; letter-spacing:-0.03em; text-transform:uppercase; line-height:1.08;">`)
	b.WriteString(escapeText(h.Title))
	b.WriteString(`</`)
	b.WriteString(titleTag)
	b.WriteString(`>
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

// writePanel writes email.Panel. Hardened mode moves each row's label and
// value from two <span>s sharing one <td> into a two-cell table row
// (pixel dossier section 4.3, point 1): Outlook Windows does not support
// display:inline-block, so a td pair aligns natively where a fixed-width
// inline-block does not. Parity mode keeps WP1's span pair, byte-for-byte.
func writePanel(b *strings.Builder, theme Theme, p doc.ResolvedPanel, hard bool) {
	b.WriteString(`<tr>
<td style="padding:24px 32px 0 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%; border:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; border-radius:4px; background-color:`)
	b.WriteString(theme.ColorPanel)
	b.WriteString(`;">
`)
	for i, row := range p.Rows {
		border := ""
		if i < len(p.Rows)-1 {
			border = ` border-bottom:1px solid ` + theme.ColorBorder + `;`
		}
		if hard {
			b.WriteString(`<tr>
<td width="108" valign="top" style="padding:16px 0 16px 20px; width:108px; color:`)
			b.WriteString(theme.ColorMuted)
			b.WriteString(`; font-family:`)
			b.WriteString(theme.FontMono)
			b.WriteString(`; font-size:11px; letter-spacing:0.06em; text-transform:uppercase;`)
			b.WriteString(border)
			b.WriteString(`">`)
			b.WriteString(escapeText(row.Label))
			b.WriteString(`</td>
<td valign="top" style="padding:16px 20px 16px 0; color:`)
			b.WriteString(theme.ColorInk)
			b.WriteString(`; font-family:`)
			b.WriteString(theme.FontSans)
			b.WriteString(`; font-size:14px;`)
			b.WriteString(border)
			b.WriteString(`">`)
			b.WriteString(escapeText(row.Value))
			b.WriteString(`</td>
</tr>
`)
			continue
		}
		b.WriteString(`<tr>
<td style="padding:16px 20px;`)
		b.WriteString(border)
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

// writeCTA writes email.CTA. Hardened mode adds mso-padding-alt to the
// face td and to the link/span itself (pixel dossier section 4.4's
// default technique, MJML's shipped pattern): Outlook gets the visual
// button box the padding gives every other client, without double
// padding. Parity mode keeps WP1's padded-<a>-only markup, byte-for-byte.
func writeCTA(b *strings.Builder, theme Theme, c doc.ResolvedCTA, hard bool) {
	b.WriteString(`<tr>
<td align="center" style="padding:28px 32px 0 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td style="border-radius:2px; background-color:`)
	b.WriteString(theme.ColorAccent)
	if hard {
		b.WriteString(`; mso-padding-alt:14px 30px;">
`)
	} else {
		b.WriteString(`;">
`)
	}
	faceStyle := "display:inline-block; padding:14px 30px; "
	if hard {
		faceStyle += "mso-padding-alt:0; "
	}
	faceStyle += "color:" + theme.ColorGround +
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

// writeNote writes email.Note. Hardened mode marks the aside structurally
// (pixel dossier section 4.7): a border-left accent bar and a tinted
// background, never a color-only cue. Parity mode keeps WP1's plain
// paragraph, byte-for-byte.
func writeNote(b *strings.Builder, theme Theme, n doc.ResolvedNote, hard bool) {
	if !hard {
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
		return
	}
	b.WriteString(`<tr>
<td style="padding:24px 32px 0 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;">
<tr>
<td style="padding:12px 16px; background-color:`)
	b.WriteString(theme.ColorPanel)
	b.WriteString(`; border-left:3px solid `)
	b.WriteString(theme.ColorAccent)
	b.WriteString(`; color:`)
	b.WriteString(theme.ColorBody)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:13px; line-height:1.5;">`)
	b.WriteString(escapeText(n.Text))
	b.WriteString(`</td>
</tr>
</table>
</td>
</tr>
`)
}

// writeDivider writes email.Divider. Hardened mode adopts the spacer
// technique (pixel dossier section 4.8): a td with font-size:0,
// line-height:0, and mso-line-height-rule:exactly pins an exact-height
// rule across every client; a plain border-top div does not. Parity mode
// keeps WP1's bare border-top div, byte-for-byte.
func writeDivider(b *strings.Builder, theme Theme, _ doc.ResolvedDivider, hard bool) {
	if !hard {
		b.WriteString(`<tr>
<td style="padding:20px 32px 0 32px;">
<div style="border-top:1px solid `)
		b.WriteString(theme.ColorBorder)
		b.WriteString(`;"></div>
</td>
</tr>
`)
		return
	}
	b.WriteString(`<tr>
<td style="padding:28px 32px 0 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;">
<tr><td style="border-top:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; font-size:0; line-height:0; mso-line-height-rule:exactly;">&nbsp;</td></tr>
</table>
</td>
</tr>
`)
}

// writeStatTable ports emailkit's StatTable.appendHTML (blocks.go), driven
// by Theme's tokens instead of package-level constants. MarkRow (1-based;
// 0 means no row is marked) selects the accent-colored row, matching
// emailkit's own MarkRow semantics (design spec section 6.5).
//
// Hardened mode applies the data-table contract (pixel dossier section
// 4.5, global invariant 7): a StatTable holds facts, so it is a real
// table, never role="presentation", and its header cells are <th
// scope="col">, not <td>. Parity mode keeps WP1's role="presentation" and
// <td> header cells, byte-for-byte.
func writeStatTable(b *strings.Builder, theme Theme, s doc.ResolvedStatTable, hard bool) {
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
	if hard {
		b.WriteString(`<table width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%; border:1px solid `)
	} else {
		b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%; border:1px solid `)
	}
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; border-radius:4px;">
`)
	if len(s.Header) > 0 {
		headerTag := "td"
		headerTagAttrs := ""
		if hard {
			headerTag = "th"
			headerTagAttrs = ` align="left" scope="col"`
		}
		b.WriteString("<tr>\n")
		for _, cell := range s.Header {
			b.WriteByte('<')
			b.WriteString(headerTag)
			b.WriteString(headerTagAttrs)
			b.WriteString(` style="padding:10px 14px; border-bottom:1px solid `)
			b.WriteString(theme.ColorBorder)
			b.WriteString(`; color:`)
			b.WriteString(theme.ColorMuted)
			b.WriteString(`; font-family:`)
			b.WriteString(theme.FontMono)
			b.WriteString(`; font-size:11px; letter-spacing:0.06em; text-transform:uppercase;">`)
			b.WriteString(escapeText(cell))
			b.WriteString("</")
			b.WriteString(headerTag)
			b.WriteString(">\n")
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
