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
	"unicode/utf8"

	"m31labs.dev/gsxmail/doc"
)

// WriteOptions configures Write's output contract (design spec section 15,
// WP5.1/WP5.2; pixel dossier section 4.2's "Shell options" surface).
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

	// Preheader, when non-empty, emits the hidden inbox-preview div as the
	// first child of <body>, in both output contracts (design spec section
	// 15, WP5.2; pixel dossier section 6.1). gsxmail.Set.Render sources it
	// from the rendered template's own Shell (design spec section 15,
	// WP5.2: preheader is authored on <email.Shell preheader={...}>, not
	// passed here directly) — a caller writing straight to WriteOptions
	// only needs this field when driving the writer without going through
	// Set.Render at all.
	Preheader string
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
	// adaptive gates the WP5.2 gsx-ink/gsx-muted class hooks (pixel dossier
	// section 5.2's adaptive style layer): parity mode never gains new
	// markup, and "none"/"locked" have no swapped-in class-driven colors to
	// hook, so the classes only appear when both conditions hold.
	adaptive := hard && theme.DarkStrategy() == "adaptive"
	var b strings.Builder
	writeHead(&b, theme, resolved.Shell, hard)
	writeBodyOpen(&b, theme, resolved.Shell, hard, adaptive, opts.Preheader)
	for _, block := range resolved.Blocks {
		writeBlock(&b, theme, block, hard, adaptive)
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
	strategy := theme.DarkStrategy()
	// metaScheme is the ColorScheme-driven pair WP5.1 shipped, unchanged
	// for "none" and "locked" (backward-compatible: a theme that never sets
	// DarkMode renders byte-identically to WP5.1). "adaptive" overrides it
	// to "light dark" regardless of ColorScheme (pixel dossier section
	// 5.2's exact snippet) — EM144 already rejects a conflicting explicit
	// ColorScheme at Load time.
	metaScheme := theme.ColorScheme
	if strategy == "adaptive" {
		metaScheme = "light dark"
	}
	if metaScheme != "" {
		b.WriteString("<meta name=\"color-scheme\" content=\"")
		b.WriteString(escapeAttr(metaScheme))
		b.WriteString("\">\n<meta name=\"supported-color-schemes\" content=\"")
		b.WriteString(escapeAttr(metaScheme))
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
`)
	writeDarkStyleLayer(b, theme, strategy)
	b.WriteString(`</style>
</head>
`)
}

// writeDarkStyleLayer appends the "locked" or "adaptive" dark-mode
// strategy's own <style> rules (pixel dossier section 5.2); "none" (the
// default) writes nothing, keeping WP5.1's <style> block byte-identical.
//
// "locked" adds one :root rule (Apple Mail 16+ honors color-scheme only on
// the root element — caniemail css-color-scheme, R2 — so this is the one
// place a locked theme can declare itself). "adaptive" adds the
// @media(prefers-color-scheme:dark) block that swaps Theme.Dark's tokens
// into the gsx-body/gsx-card/gsx-ink/gsx-muted class hooks, plus the
// [data-ogsc]/[data-ogsb] Outlook-app inversion hooks Litmus documents
// (R14) — both best-effort, never claiming control of a forced transform
// (pixel dossier section 5.1).
func writeDarkStyleLayer(b *strings.Builder, theme Theme, strategy string) {
	switch strategy {
	case "locked":
		scheme := theme.ColorScheme
		if scheme == "" {
			scheme = "dark"
		}
		b.WriteString(":root{color-scheme:")
		b.WriteString(scheme)
		b.WriteString(";}\n")
	case "adaptive":
		if theme.Dark == nil {
			// EM140 already fails Load closed on this; the writer stays
			// defensive, not exhaustive (lower.go's own stated philosophy
			// for a check-time guarantee lint already proved).
			return
		}
		d := theme.Dark
		b.WriteString("@media (prefers-color-scheme: dark) {\n.gsx-body, .gsx-card { background-color:")
		b.WriteString(d.ColorCard)
		b.WriteString(" !important; }\n.gsx-ink { color:")
		b.WriteString(d.ColorInk)
		b.WriteString(" !important; }\n.gsx-muted { color:")
		b.WriteString(d.ColorMuted)
		b.WriteString(" !important; }\n}\n[data-ogsc] .gsx-ink { color:")
		b.WriteString(d.ColorInk)
		b.WriteString(" !important; }\n[data-ogsb] .gsx-card { background-color:")
		b.WriteString(d.ColorCard)
		b.WriteString(" !important; }\n")
	}
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

func writeBodyOpen(b *strings.Builder, theme Theme, shell doc.ResolvedShell, hard, adaptive bool, preheader string) {
	if !hard {
		writeBodyOpenParity(b, theme, shell, preheader)
		return
	}
	width := strconv.Itoa(theme.CardWidth)

	b.WriteString(`<body class="gsx-body" style="margin:0; padding:0; background-color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`;">
`)
	writePreheader(b, preheader)
	b.WriteString(`<div role="article" aria-roledescription="email" aria-label="`)
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
<div`)
	b.WriteString(classAttrIf(adaptive, "gsx-ink"))
	b.WriteString(` style="color:`)
	b.WriteString(theme.ColorInk)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:19px; font-weight:800; letter-spacing:-0.02em; text-transform:uppercase;">`)
	b.WriteString(escapeText(shell.Wordmark))
	b.WriteString(`</div>
<div`)
	b.WriteString(classAttrIf(adaptive, "gsx-muted"))
	b.WriteString(` style="color:`)
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

// classAttrIf returns ` class="name"` when cond holds, or "" otherwise —
// the WP5.2 adaptive-mode gsx-ink/gsx-muted class hook (pixel dossier
// section 5.2), applied only at the Shell wordmark/tagline and Headline
// title (the highest-visibility ink/muted text): a template-wide sweep
// over every ink- or muted-colored element is a natural follow-on, not
// required for the strategy to work end to end.
func classAttrIf(cond bool, name string) string {
	if !cond {
		return ""
	}
	return ` class="` + name + `"`
}

// preheaderTargetChars is the decoded-character length the hidden
// preview-text div always reaches (pixel dossier section 6.1, citing
// react-email's shipped Preview component's padding pattern, R5): long
// enough that no supported client pulls visible body copy into the inbox
// preview line once the author's own text runs out. EM171 already rejects
// a literal preheader over this limit at Load time.
const preheaderTargetChars = 150

// writePreheader writes the hidden inbox-preview div — react-email's
// shipped suppression style set (R5) plus an alternating &nbsp;/&zwnj; pad
// tail bringing the decoded text to exactly preheaderTargetChars
// characters — as the first child of <body>, in both output contracts
// (pixel dossier section 4.2's body-opening contract; section 6.1: the
// suppression styles alone hide it, so it needs no ghost-table wrapper to
// work, and MJML's mj-preview, R6, solves the same problem without one
// either). It writes nothing when preheader is empty, so a template that
// never sets one keeps rendering byte-identically to WP5.1 in both modes.
func writePreheader(b *strings.Builder, preheader string) {
	if preheader == "" {
		return
	}
	b.WriteString(`<div style="display:none; overflow:hidden; line-height:1px; opacity:0; max-height:0; max-width:0;">`)
	b.WriteString(escapeText(preheader))
	remaining := preheaderTargetChars - utf8.RuneCountInString(preheader)
	for ; remaining >= 2; remaining -= 2 {
		b.WriteString(`&nbsp;&zwnj;`)
	}
	if remaining == 1 {
		b.WriteString(`&nbsp;`)
	}
	b.WriteString("</div>\n")
}

// writeBodyOpenParity is WP1's exact body-open writer, byte-for-byte
// (design spec section 6.4), plus the one WP5.2 addition the parity
// guarantee still allows: writePreheader, which writes nothing when
// preheader is empty. WriteOptions{Outlook: "off"} with no preheader set
// keeps pinning the original bytes (the gridiron invite DOM-parity guard);
// setting a preheader in parity mode is a deliberate, additive exception,
// since the suppression-style div needs no ghost-table wrapper to work.
func writeBodyOpenParity(b *strings.Builder, theme Theme, shell doc.ResolvedShell, preheader string) {
	width := strconv.Itoa(theme.CardWidth)

	b.WriteString(`<body style="margin:0; padding:0; background-color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`;">
`)
	writePreheader(b, preheader)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%; background-color:`)
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

func writeBlock(b *strings.Builder, theme Theme, block doc.ResolvedBlock, hard, adaptive bool) {
	switch v := block.(type) {
	case doc.ResolvedSignal:
		writeSignal(b, theme, v)
	case doc.ResolvedHeadline:
		writeHeadline(b, theme, v, hard, adaptive)
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
	case doc.ResolvedButton:
		writeButton(b, theme, v, hard)
	case doc.ResolvedColumns:
		writeColumns(b, theme, v, hard, adaptive)
	case doc.ResolvedHero:
		writeHero(b, v)
	case doc.ResolvedSpacer:
		writeSpacer(b, v)
	case doc.ResolvedBadge:
		writeBadge(b, theme, v)
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
// mode keeps WP1's <div> title, byte-for-byte. adaptive additionally marks
// the title with the gsx-ink class hook (pixel dossier section 5.2).
func writeHeadline(b *strings.Builder, theme Theme, h doc.ResolvedHeadline, hard, adaptive bool) {
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
	b.WriteString(classAttrIf(adaptive, "gsx-ink"))
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

// writeButton writes email.Button (WP5.3; pixel dossier section 4.4). The
// "primary" variant (the default) calls straight through to writeCTA,
// unchanged above, so email.CTA and email.Button variant="primary" are
// byte-identical in both output contracts by construction — CTA's own
// alias, not a parallel reimplementation that could drift. "secondary" and
// "link" are new WP5.3 shapes with no WP1 byte stream to protect, so both
// render one contract regardless of hard.
func writeButton(b *strings.Builder, theme Theme, v doc.ResolvedButton, hard bool) {
	switch v.Variant {
	case "secondary":
		writeButtonSecondary(b, theme, v, hard)
	case "link":
		writeButtonLink(b, theme, v)
	default: // "primary", and "" defensively (resolve.go already defaults it)
		writeCTA(b, theme, doc.ResolvedCTA{Label: v.Label, Href: v.Href}, hard)
	}
}

// writeButtonSecondary writes the "secondary" variant (pixel dossier
// section 4.4): a transparent face with a 1px accent border, instead of
// primary's solid accent fill. It mirrors writeCTA's own hard/parity split
// (mso-padding-alt only under the hardened contract) for the same reason:
// Outlook needs the td's own border to draw the visual box a border-only
// <a> cannot give it.
func writeButtonSecondary(b *strings.Builder, theme Theme, v doc.ResolvedButton, hard bool) {
	b.WriteString(`<tr>
<td align="center" style="padding:28px 32px 0 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td style="border-radius:2px; border:1px solid `)
	b.WriteString(theme.ColorAccent)
	if hard {
		b.WriteString(`; mso-padding-alt:13px 29px;">
`)
	} else {
		b.WriteString(`;">
`)
	}
	faceStyle := "display:inline-block; padding:13px 29px; "
	if hard {
		faceStyle += "mso-padding-alt:0; "
	}
	faceStyle += "color:" + theme.ColorAccent +
		"; font-family:" + theme.FontSans +
		"; font-size:14px; font-weight:800; letter-spacing:0.04em; text-transform:uppercase; text-decoration:none; border-radius:2px;"
	if hasSafeHrefScheme(v.Href) {
		b.WriteString(`<a href="`)
		b.WriteString(escapeAttr(v.Href))
		b.WriteString(`" style="`)
		b.WriteString(faceStyle)
		b.WriteString(`">`)
		b.WriteString(escapeText(v.Label))
		b.WriteString(`</a>
`)
	} else {
		b.WriteString(`<span style="`)
		b.WriteString(faceStyle)
		b.WriteString(`">`)
		b.WriteString(escapeText(v.Label))
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

// linkButtonDefaultWidth estimates the "link" variant's clickable width in
// pixels from its label's rune count, when a template does not set
// Button's own width attribute. gsxmail cannot measure real glyph widths
// without a font-metrics dependency the render path does not carry, so
// this is a documented approximation (README's Button section states the
// same caveat) — set width="..." explicitly for an exact click target.
func linkButtonDefaultWidth(label string) int {
	n := utf8.RuneCountInString(label)
	w := n*9 + 60
	if w < 120 {
		return 120
	}
	return w
}

// writeButtonLink writes the "link" variant: goodemailcode's full-click
// glyph-spacing technique (pixel dossier section 4.4, R11). Unlike
// primary/secondary, there is no wrapping table-cell background to give
// Outlook a clickable box; instead, an MSO-only hidden run stretched with
// a negative mso-font-width fakes the anchor's own minimum width, so the
// whole box — not just the text — is clickable there too. New in WP5.3,
// one contract regardless of hard: the technique is inert everywhere
// outside Outlook (the <i> runs sit inside "[if mso]" conditional
// comments), so there is nothing for a parity mode to strip.
func writeButtonLink(b *strings.Builder, theme Theme, v doc.ResolvedButton) {
	width := strings.TrimSpace(v.Width)
	if width == "" {
		width = strconv.Itoa(linkButtonDefaultWidth(v.Label))
	}
	b.WriteString(`<tr>
<td align="center" style="padding:28px 32px 0 32px;">
`)
	if !hasSafeHrefScheme(v.Href) {
		b.WriteString(`<span style="background-color:`)
		b.WriteString(theme.ColorAccent)
		b.WriteString(`; border-radius:2px; color:`)
		b.WriteString(theme.ColorGround)
		b.WriteString(`; display:inline-block; padding:14px 30px; font-family:`)
		b.WriteString(theme.FontSans)
		b.WriteString(`; font-size:14px; font-weight:800; letter-spacing:0.04em; text-transform:uppercase;">`)
		b.WriteString(escapeText(v.Label))
		b.WriteString(`</span>
</td>
</tr>
`)
		return
	}
	b.WriteString(`<a href="`)
	b.WriteString(escapeAttr(v.Href))
	b.WriteString(`" style="background-color:`)
	b.WriteString(theme.ColorAccent)
	b.WriteString(`; border-radius:2px; color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`; display:inline-block; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:14px; font-weight:800; letter-spacing:0.04em; line-height:44px; text-align:center; text-decoration:none; text-transform:uppercase; width:`)
	b.WriteString(width)
	b.WriteString(`px; -webkit-text-size-adjust:none;">
<!--[if mso]><i style="letter-spacing:`)
	b.WriteString(width)
	b.WriteString(`px; mso-font-width:-100%; mso-text-raise:22pt;" hidden>&nbsp;</i><![endif]-->
<span style="mso-text-raise:15pt;">`)
	b.WriteString(escapeText(v.Label))
	b.WriteString(`</span>
<!--[if mso]><i style="letter-spacing:`)
	b.WriteString(width)
	b.WriteString(`px; mso-font-width:-100%;" hidden>&nbsp;</i><![endif]-->
</a>
</td>
</tr>
`)
}

// writeColumns writes email.Columns: the fluid-hybrid contract (pixel
// dossier section 4.9) — inline-block max-width divs that stack under a
// 480px viewport with no <style> dependency, wrapped in an
// "[if mso | IE]" ghost table so Outlook, which never applies
// inline-block (caniemail css-display), gets an equivalent td-per-column
// row instead. maxWidth scales from theme.CardWidth for a two-to-four
// column row (EM176 already caps the count at Load time); the dossier's
// own worked numbers (268px per column, on a 600px card, two columns)
// fall out of this formula unchanged.
func writeColumns(b *strings.Builder, theme Theme, v doc.ResolvedColumns, hard, adaptive bool) {
	n := len(v.Columns)
	if n == 0 {
		return
	}
	maxWidth := (theme.CardWidth - 64) / n
	widthStr := strconv.Itoa(maxWidth)

	b.WriteString(`<tr>
<td style="padding:24px 32px 0 32px; font-size:0; text-align:center;">
`)
	for i, col := range v.Columns {
		if i == 0 {
			b.WriteString(`<!--[if mso | IE]><table role="presentation" width="100%" border="0" cellpadding="0" cellspacing="0"><tr><td width="`)
			b.WriteString(widthStr)
			b.WriteString(`" valign="top"><![endif]-->
`)
		} else {
			b.WriteString(`<!--[if mso | IE]></td><td width="`)
			b.WriteString(widthStr)
			b.WriteString(`" valign="top"><![endif]-->
`)
		}
		b.WriteString(`<div class="gsx-col" style="display:inline-block; width:100%; max-width:`)
		b.WriteString(widthStr)
		b.WriteString(`px; vertical-align:top; text-align:left; font-size:14px;">
`)
		writeColumnContent(b, theme, col, hard, adaptive)
		b.WriteString(`</div>
`)
	}
	b.WriteString(`<!--[if mso | IE]></td></tr></table><![endif]-->
</td>
</tr>
`)
}

// writeColumnContent writes one Column's own content: an optional image,
// an optional title, and an optional body text, each sized for a fluid-
// hybrid column's own narrower width rather than the full card (design
// note: doc.Column's own doc comment on why Column stays a leaf). The
// title carries the gsx-ink adaptive class hook under DarkMode "adaptive"
// (pixel dossier section 5.2), the same hook every other ink-colored
// heading in the card carries — Columns nests ordinary card content, so
// WP5.2's dark-mode coverage extends into it for free.
func writeColumnContent(b *strings.Builder, theme Theme, col doc.ResolvedColumn, hard, adaptive bool) {
	_ = hard
	if col.ImgSrc != "" {
		b.WriteString(`<img src="`)
		b.WriteString(escapeAttr(col.ImgSrc))
		b.WriteString(`" width="`)
		b.WriteString(col.ImgWidth)
		b.WriteString(`" height="`)
		b.WriteString(col.ImgHeight)
		b.WriteString(`" alt="`)
		b.WriteString(escapeAttr(col.ImgAlt))
		b.WriteString(`" style="display:block; width:100%; max-width:`)
		b.WriteString(col.ImgWidth)
		b.WriteString(`px; height:auto; border:0; margin-bottom:12px;">
`)
	}
	if col.Title != "" {
		b.WriteString(`<div`)
		b.WriteString(classAttrIf(adaptive, "gsx-ink"))
		b.WriteString(` style="color:`)
		b.WriteString(theme.ColorInk)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontSans)
		b.WriteString(`; font-size:15px; font-weight:800; letter-spacing:-0.01em;">`)
		b.WriteString(escapeText(col.Title))
		b.WriteString(`</div>
`)
	}
	if col.Text != "" {
		topMargin := ""
		if col.Title != "" {
			topMargin = " margin-top:6px;"
		}
		b.WriteString(`<div style="color:`)
		b.WriteString(theme.ColorBody)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontSans)
		b.WriteString(`; font-size:13px; line-height:1.5;`)
		b.WriteString(topMargin)
		b.WriteString(`">`)
		b.WriteString(escapeText(col.Text))
		b.WriteString(`</div>
`)
	}
}

// writeHero writes email.Hero: a single retina <img> at 2x asset
// resolution with display-size width/height attributes plus the
// max-width/height:auto Outlook workaround (pixel dossier section 4.10;
// R15). Hero is new in WP5.3 — there is no WP1 byte stream to protect —
// so it renders one contract regardless of the hard flag.
func writeHero(b *strings.Builder, v doc.ResolvedHero) {
	b.WriteString(`<tr>
<td style="padding:0;">
<img src="`)
	b.WriteString(escapeAttr(v.Src))
	b.WriteString(`" width="`)
	b.WriteString(v.Width)
	b.WriteString(`" height="`)
	b.WriteString(v.Height)
	b.WriteString(`" alt="`)
	b.WriteString(escapeAttr(v.Alt))
	b.WriteString(`" style="display:block; width:100%; max-width:`)
	b.WriteString(v.Width)
	b.WriteString(`px; height:auto; border:0;">
</td>
</tr>
`)
}

// writeSpacer writes email.Spacer: an exact-height gap row (pixel dossier
// section 4.8's spacer-table technique — font-size:0, line-height:0, and
// mso-line-height-rule:exactly pin the box everywhere a bare margin or an
// empty div does not). New in WP5.3, one contract regardless of hard.
func writeSpacer(b *strings.Builder, v doc.ResolvedSpacer) {
	b.WriteString(`<tr><td height="`)
	b.WriteString(v.Height)
	b.WriteString(`" style="height:`)
	b.WriteString(v.Height)
	b.WriteString(`px; font-size:0; line-height:0; mso-line-height-rule:exactly;">&nbsp;</td></tr>
`)
}

// badgeToneColor gives each Badge tone a fixed, theme-independent color
// (pixel dossier section 4.11's own worked example: a green "PAID" badge
// against the neutral Paper theme's blue accent). Status semantics should
// read the same green/amber/red regardless of a template's brand palette.
// "neutral" is the one tone that does track the active theme, using its
// muted token, since it carries no status meaning of its own to protect.
func badgeToneColor(theme Theme, tone string) string {
	switch tone {
	case "positive":
		return "#2F9E44"
	case "warning":
		return "#B76E00"
	case "critical":
		return "#C92A2A"
	default: // "neutral"
		return theme.ColorMuted
	}
}

// writeBadge writes email.Badge (pixel dossier section 4.11): a bordered,
// inline status label. Outlook drops border-radius and inline-block; the
// degrade is bordered inline text, still legible. New in WP5.3, one
// contract regardless of hard.
func writeBadge(b *strings.Builder, theme Theme, v doc.ResolvedBadge) {
	color := badgeToneColor(theme, v.Tone)
	b.WriteString(`<tr>
<td style="padding:20px 32px 0 32px;">
<span style="display:inline-block; padding:2px 8px; border:1px solid `)
	b.WriteString(color)
	b.WriteString(`; border-radius:2px; color:`)
	b.WriteString(color)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontMono)
	b.WriteString(`; font-size:10px; letter-spacing:0.06em; text-transform:uppercase;">`)
	b.WriteString(escapeText(v.Text))
	b.WriteString(`</span>
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
