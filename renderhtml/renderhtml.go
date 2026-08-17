// Package renderhtml writes a Resolved EmailDoc to the pixel-targeted HTML
// part: theme tokens become inline styles, entities decoded by gosx at
// compile time are re-escaped minimally, and attribute order follows
// source order.
//
// # Output contracts
//
// Write (and WriteWithOptions) emit hardened, bulletproof markup by
// default: role="presentation" on every layout table, an Outlook ghost
// table around the 600px card, doubled width attributes/CSS for the DPI
// fix, and every other per-component contract detail. WriteOptions.Outlook
// == "off" selects parity mode instead: the original byte stream,
// unchanged, for a consumer whose own equivalence test pins the old
// bytes (gsxmail's own gridiron invite DOM-parity guard is the example —
// see WriteOptions's doc comment).
package renderhtml

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"m31labs.dev/gsxmail/internal/doc"
)

// WriteOptions configures Write's output contract: the Shell options
// surface.
type WriteOptions struct {
	// Outlook selects the layout technique. "" and "ghost-tables" (the
	// default) emit the hardened contract: an Outlook ghost table around
	// the card, role="presentation" plus border/cellpadding/cellspacing
	// on every layout table, doubled width attributes for the Outlook DPI
	// fix, td-pair Panel rows, an <h1> Headline title, mso-padding-alt on
	// the CTA, a real StatTable data-table contract, and the border-left
	// Note / spacer-technique Divider. "off" emits the original byte
	// stream unchanged — the parity mode a consumer's own byte- or
	// DOM-equivalence test can pin.
	Outlook string

	// Preheader, when non-empty, emits the hidden inbox-preview div as the
	// first child of <body>, in both output contracts. gsxmail.Set.Render
	// sources it from the rendered template's own Shell (preheader is
	// authored on <email.Shell preheader={...}>, not
	// passed here directly) — a caller writing straight to WriteOptions
	// only needs this field when driving the writer without going through
	// Set.Render at all.
	Preheader string
}

// hardened reports whether opts selects the hardened output contract (the
// default) rather than parity mode.
func (opts WriteOptions) hardened() bool {
	return opts.Outlook != "off"
}

// RenderFinding is one render-time finding WriteWithOptions produces
// alongside the rendered HTML string — today, only EM110: a CTA or
// Button href that fails hasSafeHrefScheme drops the link
// and renders the label alone, visibly (a mid-send loop must not die for
// one bad optional link), but no longer silently. gsxmail.Set.Render
// copies each RenderFinding into the returned Parts.Diagnostics.
type RenderFinding struct {
	Code    string
	Message string
}

// Write renders resolved to a full HTML document string using theme, in
// the hardened output contract (WriteOptions{}'s default).
func Write(resolved *doc.Resolved, theme Theme) (string, []RenderFinding) {
	return WriteWithOptions(resolved, theme, WriteOptions{})
}

// WriteWithOptions is Write with an explicit WriteOptions (see the
// package doc and WriteOptions for the two output contracts).
func WriteWithOptions(resolved *doc.Resolved, theme Theme, opts WriteOptions) (string, []RenderFinding) {
	hard := opts.hardened()
	// adaptive gates every adaptive-mode class-hook site: parity mode
	// never gains new markup, and
	// "none"/"locked" have no swapped-in class-driven colors to hook, so
	// the classes only appear when both conditions hold.
	adaptive := hard && theme.DarkStrategy() == "adaptive"
	var b strings.Builder
	var findings []RenderFinding
	writeHead(&b, theme, resolved.Shell, hard)
	writeBodyOpen(&b, theme, resolved.Shell, hard, adaptive, opts.Preheader, &findings)
	for _, block := range resolved.Blocks {
		writeBlock(&b, theme, block, hard, adaptive, &findings)
	}
	writeBodyClose(&b, hard)
	return b.String(), findings
}

// writeHead's hardened <html> tag always writes dir="ltr" (m18, launch-
// gate findings): Shell.Lang is a plain string a template sets to any
// BCP-47 tag, including an RTL language's own ("ar", "he", ...), but
// gsxmail does not read it to pick a direction — it has no RTL layout
// support at all today (mirror padding/alignment, bidi-safe number and
// punctuation handling), so hardcoding "ltr" is the honest choice: it
// never lies about a direction gsxmail did not actually lay out for,
// even when Lang itself names an RTL language. See the README's own
// note on this boundary.
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
	// metaScheme is the ColorScheme-driven pair, unchanged for "none" and
	// "locked" (backward-compatible: a theme that never sets DarkMode
	// renders byte-identically to the original output). "adaptive"
	// overrides it to "light dark" regardless of ColorScheme — EM144
	// already rejects a conflicting explicit
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
// strategy's own <style> rules; "none" (the default) writes nothing,
// keeping the original <style> block byte-identical.
//
// "locked" adds one :root rule (Apple Mail 16+ honors color-scheme only on
// the root element — caniemail css-color-scheme, R2 — so this is the one
// place a locked theme can declare itself). "adaptive" adds the
// @media(prefers-color-scheme:dark) block that swaps every one of
// Theme.Dark's nine tokens into its own class hook, plus the best-effort
// [data-ogsc]/[data-ogsb] Outlook-app inversion hooks Litmus documents
// (R14) for every one of them — both best-effort, never claiming control
// of a forced transform.
//
// The eleven class hooks, and the Dark token each swaps in:
//
//   - gsx-body (background-color): Dark.ColorGround — the page background
//     outside the card. A light Theme.ColorGround/ColorCard pair that
//     differ (the shipped default does) needs its own hook here,
//     separate from gsx-card's own rule, or the page background outside
//     the card renders the wrong tone under "adaptive".
//   - gsx-card (background-color): Dark.ColorCard — the card itself.
//   - gsx-ink (color): Dark.ColorInk — headline/wordmark text.
//   - gsx-muted (color, border-color): Dark.ColorMuted — mono labels; the
//     border-color half is Badge's neutral tone, which borders and colors
//     the same span with the same token (a no-op on every gsx-muted site
//     that has no border to begin with).
//   - gsx-copy (color): Dark.ColorBody — body copy, so it never renders
//     at a failing contrast ratio against the dark card with no hook at
//     all.
//   - gsx-faint (color): Dark.ColorFaint — the footer's fine print.
//   - gsx-panel (background-color): Dark.ColorPanel — Panel/Note/marked
//     StatTable row backgrounds.
//   - gsx-border (border-color): Dark.ColorBorder — every structural rule
//     and cell border.
//   - gsx-accent (color): Dark.ColorAccent — Signal text, PickList
//     numerals, a marked StatTable row, Button's secondary-variant text.
//   - gsx-accent-bg (background-color): Dark.ColorAccent — CTA/Button
//     primary and link faces.
//   - gsx-accent-border (border-color): Dark.ColorAccent — the Shell icon
//     box, Button's secondary variant, Note's border-left bar.
//   - gsx-ground (color): Dark.ColorGround — the primary/link button
//     face's own inverse text color (design intent: it always tracks
//     whichever Ground the active presentation uses, light or dark, the
//     same way it already does in every non-adaptive render).
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
		b.WriteString("@media (prefers-color-scheme: dark) {\n")
		b.WriteString(".gsx-body { background-color:" + d.ColorGround + " !important; }\n")
		b.WriteString(".gsx-card { background-color:" + d.ColorCard + " !important; }\n")
		b.WriteString(".gsx-ink { color:" + d.ColorInk + " !important; }\n")
		b.WriteString(".gsx-muted { color:" + d.ColorMuted + " !important; border-color:" + d.ColorMuted + " !important; }\n")
		b.WriteString(".gsx-copy { color:" + d.ColorBody + " !important; }\n")
		b.WriteString(".gsx-faint { color:" + d.ColorFaint + " !important; }\n")
		b.WriteString(".gsx-panel { background-color:" + d.ColorPanel + " !important; }\n")
		b.WriteString(".gsx-border { border-color:" + d.ColorBorder + " !important; }\n")
		b.WriteString(".gsx-accent { color:" + d.ColorAccent + " !important; }\n")
		b.WriteString(".gsx-accent-bg { background-color:" + d.ColorAccent + " !important; }\n")
		b.WriteString(".gsx-accent-border { border-color:" + d.ColorAccent + " !important; }\n")
		b.WriteString(".gsx-ground { color:" + d.ColorGround + " !important; }\n")
		b.WriteString("}\n")
		b.WriteString("[data-ogsc] .gsx-ink { color:" + d.ColorInk + " !important; }\n")
		b.WriteString("[data-ogsc] .gsx-muted { color:" + d.ColorMuted + " !important; }\n")
		b.WriteString("[data-ogsc] .gsx-copy { color:" + d.ColorBody + " !important; }\n")
		b.WriteString("[data-ogsc] .gsx-faint { color:" + d.ColorFaint + " !important; }\n")
		b.WriteString("[data-ogsc] .gsx-accent { color:" + d.ColorAccent + " !important; }\n")
		b.WriteString("[data-ogsc] .gsx-ground { color:" + d.ColorGround + " !important; }\n")
		b.WriteString("[data-ogsb] .gsx-body { background-color:" + d.ColorGround + " !important; }\n")
		b.WriteString("[data-ogsb] .gsx-card { background-color:" + d.ColorCard + " !important; }\n")
		b.WriteString("[data-ogsb] .gsx-panel { background-color:" + d.ColorPanel + " !important; }\n")
		b.WriteString("[data-ogsb] .gsx-accent-bg { background-color:" + d.ColorAccent + " !important; }\n")
	}
}

// writeHeadParity is the original head writer, byte-for-byte. It stays
// untouched so WriteOptions{Outlook:"off"} keeps
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

func writeBodyOpen(b *strings.Builder, theme Theme, shell doc.ResolvedShell, hard, adaptive bool, preheader string, findings *[]RenderFinding) {
	if !hard {
		writeBodyOpenParity(b, theme, shell, preheader, findings)
		return
	}
	width := strconv.Itoa(theme.CardWidth)

	b.WriteString(`<body class="gsx-body" style="margin:0; padding:0; background-color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`;">
`)
	writePreheader(b, preheader, findings)
	// dir="ltr" here is the same hardcoded, honest default writeHead's
	// own doc comment explains (m18) — this article wrapper's own
	// direction always matches the <html> tag's.
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
	b.WriteString(`" cellpadding="0" cellspacing="0" border="0" class="gsx-card`)
	if adaptive {
		b.WriteString(` gsx-border`)
	}
	b.WriteString(`" style="width:`)
	b.WriteString(width)
	b.WriteString(`px; max-width:`)
	b.WriteString(width)
	b.WriteString(`px; background-color:`)
	b.WriteString(theme.ColorCard)
	b.WriteString(`; border:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; border-radius:4px;">
<tr>
<td`)
	b.WriteString(classAttrIf(adaptive, "gsx-border"))
	b.WriteString(` style="padding:24px 32px 20px 32px; border-bottom:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td width="44" valign="middle" style="width:44px; padding-right:12px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0"`)
	b.WriteString(classAttrIf(adaptive, "gsx-accent-border"))
	b.WriteString(` style="border:1px solid `)
	b.WriteString(theme.ColorAccent)
	b.WriteString(`; border-radius:2px;"><tr><td`)
	b.WriteString(classAttrIf(adaptive, "gsx-accent"))
	b.WriteString(` style="width:40px; height:40px; text-align:center; vertical-align:middle; color:`)
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
// the adaptive-mode class hook, applied at every site that writes one of
// Theme's nine color tokens inline — see writeDarkStyleLayer's own doc
// comment for the full class-to-token table.
func classAttrIf(cond bool, name string) string {
	if !cond {
		return ""
	}
	return ` class="` + name + `"`
}

// classAttrIf2 is classAttrIf for the handful of sites that need two
// adaptive class hooks on the same element at once (a StatTable cell that
// is both bordered and, when marked, accent-colored, for instance).
// Either name may be empty, in which case it is skipped.
func classAttrIf2(cond bool, name1, name2 string) string {
	if !cond {
		return ""
	}
	var names []string
	if name1 != "" {
		names = append(names, name1)
	}
	if name2 != "" {
		names = append(names, name2)
	}
	if len(names) == 0 {
		return ""
	}
	return ` class="` + strings.Join(names, " ") + `"`
}

// preheaderTargetChars is the decoded-character length the hidden
// preview-text div always reaches (react-email's shipped Preview
// component's own padding pattern, R5): long
// enough that no supported client pulls visible body copy into the inbox
// preview line once the author's own text runs out. EM171 already rejects
// a literal preheader over this limit at Load time; a dynamic
// {expression} preheader cannot be checked until Render, so writePreheader
// truncates to this same limit there instead.
const preheaderTargetChars = 150

// preheaderPadPair and preheaderPadNBSP are writePreheader's own pad-tail
// characters, written as raw UTF-8 instead of the
// "&nbsp;&zwnj;"/"&nbsp;" entity spelling: the document already declares
// UTF-8 (<meta charset="utf-8">, writeHead's own first line), so a raw
// U+00A0 (non-breaking space) plus U+200C (zero-width non-joiner) costs 5
// bytes per pair instead of 12, and a lone trailing NBSP costs 2 bytes
// instead of 6 — the same two decoded characters, cheaper on the wire,
// at up to 74 pairs per preheader. This is purely a byte-cost choice for
// gsxmail's own synthesized padding, not a change to escapeText's own
// entity-spelled NBSP in real interpolated text (that rule protects diff
// visibility for props-driven content; this padding is never diffed
// against anything).
const preheaderPadPair = " ‌"
const preheaderPadNBSP = " "

// writePreheader writes the hidden inbox-preview div — react-email's
// shipped suppression style set (R5) plus an alternating &nbsp;/&zwnj; pad
// tail bringing the decoded text to exactly preheaderTargetChars
// characters — as the first child of <body>, in both output contracts.
// The suppression styles alone hide it, so it needs no ghost-table
// wrapper to work, and MJML's mj-preview, R6, solves the same problem
// without one either. It writes nothing when preheader is empty, so a
// template that never sets one keeps rendering byte-identically in both
// modes.
//
// A preheader over preheaderTargetChars runes is truncated to exactly
// that many, and appends an EM200 RenderFinding to findings: EM171
// already rejects a literal (static) preheader this long at
// Load time, but a dynamic {expression} preheader's own length is not
// known until a real props value resolves it, so this is that same
// guarantee's render-time backstop, visible in Parts.Diagnostics rather
// than silently overflowing the inbox-preview contract.
func writePreheader(b *strings.Builder, preheader string, findings *[]RenderFinding) {
	if preheader == "" {
		return
	}
	runes := []rune(preheader)
	if len(runes) > preheaderTargetChars {
		if findings != nil {
			*findings = append(*findings, RenderFinding{
				Code: "EM200",
				Message: fmt.Sprintf(
					"preheader is %d characters; truncated to %d (the emitted block pads to exactly %d)",
					len(runes), preheaderTargetChars, preheaderTargetChars),
			})
		}
		preheader = string(runes[:preheaderTargetChars])
	}
	b.WriteString(`<div style="display:none; overflow:hidden; line-height:1px; opacity:0; max-height:0; max-width:0;">`)
	b.WriteString(escapeText(preheader))
	remaining := preheaderTargetChars - utf8.RuneCountInString(preheader)
	for ; remaining >= 2; remaining -= 2 {
		b.WriteString(preheaderPadPair)
	}
	if remaining == 1 {
		b.WriteString(preheaderPadNBSP)
	}
	b.WriteString("</div>\n")
}

// writeBodyOpenParity is the original body-open writer, byte-for-byte,
// plus the one addition the parity guarantee still allows:
// writePreheader, which writes nothing when
// preheader is empty. WriteOptions{Outlook: "off"} with no preheader set
// keeps pinning the original bytes (the gridiron invite DOM-parity guard);
// setting a preheader in parity mode is a deliberate, additive exception,
// since the suppression-style div needs no ghost-table wrapper to work.
func writeBodyOpenParity(b *strings.Builder, theme Theme, shell doc.ResolvedShell, preheader string, findings *[]RenderFinding) {
	width := strconv.Itoa(theme.CardWidth)

	b.WriteString(`<body style="margin:0; padding:0; background-color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`;">
`)
	writePreheader(b, preheader, findings)
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

func writeBlock(b *strings.Builder, theme Theme, block doc.ResolvedBlock, hard, adaptive bool, findings *[]RenderFinding) {
	switch v := block.(type) {
	case doc.ResolvedSignal:
		writeSignal(b, theme, v, adaptive)
	case doc.ResolvedHeadline:
		writeHeadline(b, theme, v, hard, adaptive)
	case doc.ResolvedPanel:
		writePanel(b, theme, v, hard, adaptive)
	case doc.ResolvedCTA:
		writeCTA(b, theme, v, hard, adaptive, "email.CTA", findings)
	case doc.ResolvedPickList:
		writePickList(b, theme, v, adaptive)
	case doc.ResolvedFooter:
		writeFooter(b, theme, v, adaptive)
	case doc.ResolvedNote:
		writeNote(b, theme, v, hard, adaptive)
	case doc.ResolvedDivider:
		writeDivider(b, theme, v, hard, adaptive)
	case doc.ResolvedStatTable:
		writeStatTable(b, theme, v, hard, adaptive)
	case doc.ResolvedCustom:
		writeCustomNode(b, v.Root)
	case doc.ResolvedButton:
		writeButton(b, theme, v, hard, adaptive, findings)
	case doc.ResolvedColumns:
		writeColumns(b, theme, v, hard, adaptive)
	case doc.ResolvedHero:
		writeHero(b, v)
	case doc.ResolvedSpacer:
		writeSpacer(b, v)
	case doc.ResolvedBadge:
		writeBadge(b, theme, v, adaptive)
	}
}

func writeSignal(b *strings.Builder, theme Theme, s doc.ResolvedSignal, adaptive bool) {
	b.WriteString(`<tr>
<td style="padding:20px 32px 0 32px;">
<div`)
	b.WriteString(classAttrIf(adaptive, "gsx-accent"))
	b.WriteString(` style="color:`)
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
// a semantic <h1> with margins zeroed: screen-reader navigation gains a
// heading at no visual cost. Parity mode keeps the original <div> title,
// byte-for-byte. adaptive additionally marks the title with gsx-ink and
// the lede with gsx-copy.
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
<div`)
	b.WriteString(classAttrIf(adaptive, "gsx-copy"))
	b.WriteString(` style="color:`)
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
// value from two <span>s sharing one <td> into a two-cell table row:
// Outlook Windows does not support display:inline-block, so a td pair
// aligns natively where a fixed-width inline-block does not. Parity
// mode keeps the original span pair,
// byte-for-byte. adaptive marks the outer table (gsx-border, gsx-panel),
// each row's border, each label (gsx-muted), and each value (gsx-ink).
func writePanel(b *strings.Builder, theme Theme, p doc.ResolvedPanel, hard, adaptive bool) {
	b.WriteString(`<tr>
<td style="padding:24px 32px 0 32px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"`)
	b.WriteString(classAttrIf2(adaptive, "gsx-border", "gsx-panel"))
	b.WriteString(` style="width:100%; border:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; border-radius:4px; background-color:`)
	b.WriteString(theme.ColorPanel)
	b.WriteString(`;">
`)
	for i, row := range p.Rows {
		border := ""
		borderClass := ""
		if i < len(p.Rows)-1 {
			border = ` border-bottom:1px solid ` + theme.ColorBorder + `;`
			borderClass = "gsx-border"
		}
		if hard {
			b.WriteString(`<tr>
<td width="108" valign="top"`)
			b.WriteString(classAttrIf2(adaptive, "gsx-muted", borderClass))
			b.WriteString(` style="padding:16px 0 16px 20px; width:108px; color:`)
			b.WriteString(theme.ColorMuted)
			b.WriteString(`; font-family:`)
			b.WriteString(theme.FontMono)
			b.WriteString(`; font-size:11px; letter-spacing:0.06em; text-transform:uppercase;`)
			b.WriteString(border)
			b.WriteString(`">`)
			b.WriteString(escapeText(row.Label))
			b.WriteString(`</td>
<td valign="top"`)
			b.WriteString(classAttrIf2(adaptive, "gsx-ink", borderClass))
			b.WriteString(` style="padding:16px 20px 16px 0; color:`)
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
// face td and to the link/span itself — MJML's own shipped pattern:
// Outlook gets the visual button box the padding gives every other
// client, without double padding. Parity mode keeps the original
// padded-<a>-only markup,
// byte-for-byte. adaptive marks the face td (gsx-accent-bg) and the
// label itself (gsx-ground, since the label's own color is
// Theme.ColorGround — the button's inverse-of-fill text token). A
// disallowed href drops the link (the label still renders, unclickable)
// and appends an EM110 RenderFinding to findings, naming component
// ("email.CTA" or "email.Button", whichever actually called this) —
// visible, not silent.
func writeCTA(b *strings.Builder, theme Theme, c doc.ResolvedCTA, hard, adaptive bool, component string, findings *[]RenderFinding) {
	b.WriteString(`<tr>
<td align="center" style="padding:28px 32px 0 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td`)
	b.WriteString(classAttrIf(adaptive, "gsx-accent-bg"))
	b.WriteString(` style="border-radius:2px; background-color:`)
	b.WriteString(theme.ColorAccent)
	if hard {
		b.WriteString(`; mso-padding-alt:14px 30px;">
`)
	} else {
		b.WriteString(`;">
`)
	}
	faceClass := classAttrIf(adaptive, "gsx-ground")
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
		b.WriteString(`"`)
		b.WriteString(faceClass)
		b.WriteString(` style="`)
		b.WriteString(faceStyle)
		b.WriteString(`">`)
		b.WriteString(escapeText(c.Label))
		b.WriteString(`</a>
`)
	} else {
		appendHrefRejected(findings, component, c.Href)
		b.WriteString(`<span`)
		b.WriteString(faceClass)
		b.WriteString(` style="`)
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

// appendHrefRejected records an EM110 RenderFinding for one CTA/Button
// element whose Href failed hasSafeHrefScheme: the
// writer already drops the link and renders the label alone,
// unclickable — this makes that drop visible in Parts.Diagnostics
// instead of silent. findings may be nil (a caller — today, only tests —
// driving the writer directly with no interest in findings); appending
// to a nil *[]RenderFinding is a deliberate no-op, not a panic.
func appendHrefRejected(findings *[]RenderFinding, component, href string) {
	if findings == nil {
		return
	}
	*findings = append(*findings, RenderFinding{
		Code: "EM110",
		Message: fmt.Sprintf(
			"%s href %q uses a disallowed scheme (allowed: https, http, mailto); rendering the label without a link",
			component, href),
	})
}

// writeButton writes email.Button. The
// "primary" variant (the default) calls straight through to writeCTA,
// unchanged above, so email.CTA and email.Button variant="primary" are
// byte-identical in both output contracts by construction — CTA's own
// alias, not a parallel reimplementation that could drift. "secondary" and
// "link" have no original byte stream to protect, so both
// render one contract regardless of hard.
func writeButton(b *strings.Builder, theme Theme, v doc.ResolvedButton, hard, adaptive bool, findings *[]RenderFinding) {
	switch v.Variant {
	case "secondary":
		writeButtonSecondary(b, theme, v, hard, adaptive, findings)
	case "link":
		writeButtonLink(b, theme, v, adaptive, findings)
	default: // "primary", and "" defensively (resolve.go already defaults it)
		writeCTA(b, theme, doc.ResolvedCTA{Label: v.Label, Href: v.Href}, hard, adaptive, "email.Button", findings)
	}
}

// writeButtonSecondary writes the "secondary" variant: a transparent
// face with a 1px accent border, instead of primary's solid accent
// fill. It mirrors writeCTA's own hard/parity split
// (mso-padding-alt only under the hardened contract) for the same reason:
// Outlook needs the td's own border to draw the visual box a border-only
// <a> cannot give it. adaptive marks the face td's border (gsx-accent-
// border) and the label's own accent text (gsx-accent). A disallowed
// href appends an EM110 RenderFinding, same as writeCTA.
func writeButtonSecondary(b *strings.Builder, theme Theme, v doc.ResolvedButton, hard, adaptive bool, findings *[]RenderFinding) {
	b.WriteString(`<tr>
<td align="center" style="padding:28px 32px 0 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td`)
	b.WriteString(classAttrIf(adaptive, "gsx-accent-border"))
	b.WriteString(` style="border-radius:2px; border:1px solid `)
	b.WriteString(theme.ColorAccent)
	if hard {
		b.WriteString(`; mso-padding-alt:13px 29px;">
`)
	} else {
		b.WriteString(`;">
`)
	}
	faceClass := classAttrIf(adaptive, "gsx-accent")
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
		b.WriteString(`"`)
		b.WriteString(faceClass)
		b.WriteString(` style="`)
		b.WriteString(faceStyle)
		b.WriteString(`">`)
		b.WriteString(escapeText(v.Label))
		b.WriteString(`</a>
`)
	} else {
		appendHrefRejected(findings, "email.Button", v.Href)
		b.WriteString(`<span`)
		b.WriteString(faceClass)
		b.WriteString(` style="`)
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

// linkGlyphWidthPx is the per-rune pixel estimate both
// linkButtonDefaultWidth and linkButtonSpacerWidth reuse: gsxmail cannot
// measure real glyph widths without a font-metrics dependency the render
// path does not carry, so this is a documented approximation (README's
// Button section states the same caveat) — set width="..." explicitly
// for an exact click target.
const linkGlyphWidthPx = 9

// linkButtonDefaultWidth estimates the "link" variant's clickable width in
// pixels from its label's rune count, when a template does not set
// Button's own width attribute.
func linkButtonDefaultWidth(label string) int {
	n := utf8.RuneCountInString(label)
	w := n*linkGlyphWidthPx + 60
	if w < 120 {
		return 120
	}
	return w
}

// linkButtonSpacerWidth computes each of writeButtonLink's two balancing
// invisible-run widths: half of whatever room remains
// once the label's own estimated rendered width (linkGlyphWidthPx per
// rune, the same estimate linkButtonDefaultWidth uses for the whole
// button) is subtracted from widthPx, floored at 0. Outlook lays the
// anchor's own content out left to right with no text-align effect on
// the hidden runs themselves, so one full-width run before the label
// pushes the label to the right edge of its own click box
// instead of centering it; splitting that width in half and running one
// spacer on each side keeps the label centered the way `text-align:
// center` already centers it in every other client.
func linkButtonSpacerWidth(widthPx int, label string) int {
	n := utf8.RuneCountInString(label)
	spacer := (widthPx - n*linkGlyphWidthPx) / 2
	if spacer < 0 {
		return 0
	}
	return spacer
}

// writeButtonLink writes the "link" variant: goodemailcode's full-click
// glyph-spacing technique (R11). Unlike
// primary/secondary, there is no wrapping table-cell background to give
// Outlook a clickable box; instead, two MSO-only hidden runs stretched
// with a negative mso-font-width fake the anchor's own minimum width, so
// the whole box — not just the text — is clickable there too. Each run's
// own width is linkButtonSpacerWidth's own balancing half, not the full
// button width: a single full-width run pushed the
// label to the right edge of its own click box in Outlook, instead of
// centering it the way text-align:center already centers it everywhere
// else. One contract regardless of hard: the technique is
// inert everywhere outside Outlook (the <i> runs sit inside "[if mso]"
// conditional comments), so there is nothing for a parity mode to strip.
// adaptive marks the face (gsx-accent-bg, gsx-ground).
func writeButtonLink(b *strings.Builder, theme Theme, v doc.ResolvedButton, adaptive bool, findings *[]RenderFinding) {
	widthStr := strings.TrimSpace(v.Width)
	widthPx := 0
	if widthStr == "" {
		widthPx = linkButtonDefaultWidth(v.Label)
		widthStr = strconv.Itoa(widthPx)
	} else if n, err := strconv.Atoi(widthStr); err == nil {
		// doc.Resolve's resolvePositiveInt already guarantees a non-empty
		// Width is a positive decimal integer (B1), so this Atoi only
		// fails defensively (a raw ResolvedButton built outside Resolve,
		// as a test fixture might): the spacer floors at 0 either way.
		widthPx = n
	}
	spacer := strconv.Itoa(linkButtonSpacerWidth(widthPx, v.Label))
	width := widthStr
	faceClass := classAttrIf2(adaptive, "gsx-accent-bg", "gsx-ground")
	b.WriteString(`<tr>
<td align="center" style="padding:28px 32px 0 32px;">
`)
	if !hasSafeHrefScheme(v.Href) {
		appendHrefRejected(findings, "email.Button", v.Href)
		b.WriteString(`<span`)
		b.WriteString(faceClass)
		b.WriteString(` style="background-color:`)
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
	b.WriteString(`"`)
	b.WriteString(faceClass)
	b.WriteString(` style="background-color:`)
	b.WriteString(theme.ColorAccent)
	b.WriteString(`; border-radius:2px; color:`)
	b.WriteString(theme.ColorGround)
	b.WriteString(`; display:inline-block; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:14px; font-weight:800; letter-spacing:0.04em; line-height:44px; text-align:center; text-decoration:none; text-transform:uppercase; width:`)
	b.WriteString(escapeAttr(width))
	b.WriteString(`px; -webkit-text-size-adjust:none;">
<!--[if mso]><i style="letter-spacing:`)
	b.WriteString(escapeAttr(spacer))
	b.WriteString(`px; mso-font-width:-100%; mso-text-raise:22pt;" hidden>&nbsp;</i><![endif]-->
<span style="mso-text-raise:15pt;">`)
	b.WriteString(escapeText(v.Label))
	b.WriteString(`</span>
<!--[if mso]><i style="letter-spacing:`)
	b.WriteString(escapeAttr(spacer))
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
// title carries the gsx-ink adaptive class hook and the body text carries
// gsx-copy, the same hooks every other ink- and body-
// colored element in the card carries — Columns nests ordinary card
// content, so the adaptive dark-mode coverage extends into it for free.
func writeColumnContent(b *strings.Builder, theme Theme, col doc.ResolvedColumn, hard, adaptive bool) {
	_ = hard
	if col.ImgSrc != "" {
		b.WriteString(`<img src="`)
		b.WriteString(escapeAttr(col.ImgSrc))
		b.WriteString(`" width="`)
		b.WriteString(escapeAttr(col.ImgWidth))
		b.WriteString(`" height="`)
		b.WriteString(escapeAttr(col.ImgHeight))
		b.WriteString(`" alt="`)
		b.WriteString(escapeAttr(col.ImgAlt))
		b.WriteString(`" style="display:block; width:100%; max-width:`)
		b.WriteString(escapeAttr(col.ImgWidth))
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
		b.WriteString(`<div`)
		b.WriteString(classAttrIf(adaptive, "gsx-copy"))
		b.WriteString(` style="color:`)
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
// max-width/height:auto Outlook workaround (R15). Hero has no original
// byte stream to protect, so it renders one contract regardless of the
// hard flag. It carries no
// theme color of its own, so it needs no adaptive class hook.
func writeHero(b *strings.Builder, v doc.ResolvedHero) {
	b.WriteString(`<tr>
<td style="padding:0;">
<img src="`)
	b.WriteString(escapeAttr(v.Src))
	b.WriteString(`" width="`)
	b.WriteString(escapeAttr(v.Width))
	b.WriteString(`" height="`)
	b.WriteString(escapeAttr(v.Height))
	b.WriteString(`" alt="`)
	b.WriteString(escapeAttr(v.Alt))
	b.WriteString(`" style="display:block; width:100%; max-width:`)
	b.WriteString(escapeAttr(v.Width))
	b.WriteString(`px; height:auto; border:0;">
</td>
</tr>
`)
}

// writeSpacer writes email.Spacer: an exact-height gap row (a spacer-table
// technique — font-size:0, line-height:0, and
// mso-line-height-rule:exactly pin the box everywhere a bare margin or an
// empty div does not). One contract regardless of hard. It
// carries no theme color of its own, so it needs no adaptive class hook.
func writeSpacer(b *strings.Builder, v doc.ResolvedSpacer) {
	b.WriteString(`<tr><td height="`)
	b.WriteString(escapeAttr(v.Height))
	b.WriteString(`" style="height:`)
	b.WriteString(escapeAttr(v.Height))
	b.WriteString(`px; font-size:0; line-height:0; mso-line-height-rule:exactly;">&nbsp;</td></tr>
`)
}

// badgeToneVariant is one status tone's two fixed hex values: light is
// for a light card (verified >=4.5:1
// against #FFFFFF), dark is for a dark card (verified >=4.5:1 against
// #101611, Terminal theme's own ColorCard). One flat hex cannot clear
// 4.5:1 against both: the WCAG 2 contrast formula has no solution that
// does (a color light enough to read on #101611 is, by construction, too
// light to also clear 4.5:1 on white, and vice versa) — computed
// directly, not assumed, so each tone carries the two nearest-hue values
// that do clear their own card.
type badgeToneVariant struct {
	light, dark string
}

// badgeTones is the closed set of Badge's three status tones. Status
// semantics should read the same green/amber/red regardless of a
// template's brand palette; only the light/dark variant selection tracks
// the active theme's own card (badgeToneColor), never the theme's other
// tokens. "neutral" is not here: it tracks the active theme's own muted
// token instead, since it carries no status meaning of its own to
// protect (badgeToneColor's own default case).
var badgeTones = map[string]badgeToneVariant{
	"positive": {light: "#28873A", dark: "#2F9E44"}, // vs #FFFFFF 4.55:1, vs #101611 5.32:1
	"warning":  {light: "#AA6600", dark: "#B76E00"}, // vs #FFFFFF 4.56:1, vs #101611 4.58:1
	"critical": {light: "#C92A2A", dark: "#DA4E4E"}, // vs #FFFFFF 5.46:1, vs #101611 4.53:1
}

// BadgeToneColors returns the status-tone hex badgeToneColor would select
// for each of Badge's three status tones under theme's own ColorCard —
// "neutral" is deliberately absent, since it always tracks
// theme.ColorMuted rather than one of the fixed pairs, and lint's own
// checkPalette (EM141, M2) already checks ColorMuted against ColorCard
// through Theme's regular token set. Exported so package lint can extend
// EM141 to cover badge tones too, without lint reimplementing
// badgeToneVariant's own light/dark table.
func BadgeToneColors(theme Theme) map[string]string {
	out := make(map[string]string, len(badgeTones))
	for tone := range badgeTones {
		out[tone] = badgeToneColor(theme, tone)
	}
	return out
}

// badgeToneColor gives each Badge tone a color that clears WCAG AA
// (4.5:1) against theme.ColorCard, the card it actually renders on: a
// light-card and a dark-card variant per tone (badgeToneVariant),
// selected by cardIsDark. "neutral" is the one tone that does track the
// active theme's own token (ColorMuted) rather than a fixed pair, since
// it carries no status meaning of its own to protect — so it is also the
// one tone with an adaptive class hook (writeBadge).
func badgeToneColor(theme Theme, tone string) string {
	variant, ok := badgeTones[tone]
	if !ok { // "neutral", and "" defensively (resolve.go already defaults it)
		return theme.ColorMuted
	}
	if cardIsDark(theme.ColorCard) {
		return variant.dark
	}
	return variant.light
}

// writeBadge writes email.Badge: a bordered,
// inline status label. Outlook drops border-radius and inline-block; the
// degrade is bordered inline text, still legible. One
// contract regardless of hard. adaptive marks a "neutral" badge with
// gsx-muted (both its border and its text share theme.ColorMuted, and
// the gsx-muted rule swaps both); the three fixed-hex
// tones are deliberately theme-independent, so they carry no hook.
func writeBadge(b *strings.Builder, theme Theme, v doc.ResolvedBadge, adaptive bool) {
	color := badgeToneColor(theme, v.Tone)
	b.WriteString(`<tr>
<td style="padding:20px 32px 0 32px;">
<span`)
	b.WriteString(classAttrIf(adaptive && v.Tone == "neutral", "gsx-muted"))
	b.WriteString(` style="display:inline-block; padding:2px 8px; border:1px solid `)
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

// writePickList writes email.PickList. adaptive marks the title
// (gsx-muted), each item's numeral (gsx-accent), and each item's own text
// (gsx-copy).
func writePickList(b *strings.Builder, theme Theme, p doc.ResolvedPickList, adaptive bool) {
	b.WriteString(`<tr>
<td style="padding:28px 32px 0 32px;">
`)
	if p.Title != "" {
		b.WriteString(`<div`)
		b.WriteString(classAttrIf(adaptive, "gsx-muted"))
		b.WriteString(` style="color:`)
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
		b.WriteString(`<tr><td`)
		b.WriteString(classAttrIf(adaptive, "gsx-copy"))
		b.WriteString(` style="padding:5px 0; color:`)
		b.WriteString(theme.ColorBody)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontSans)
		b.WriteString(`; font-size:14px;"><span`)
		b.WriteString(classAttrIf(adaptive, "gsx-accent"))
		b.WriteString(` style="color:`)
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

// writeNote writes email.Note. Hardened mode marks the aside
// structurally: a border-left accent bar and a tinted
// background, never a color-only cue. Parity mode keeps the original plain
// paragraph, byte-for-byte — and, since adaptive is only ever true when
// hard is (WriteWithOptions's own adaptive computation), the parity
// branch below needs no class hooks of its own. Hardened mode's adaptive
// hooks: gsx-panel (background), gsx-accent-border (the left bar), and
// gsx-copy (text).
func writeNote(b *strings.Builder, theme Theme, n doc.ResolvedNote, hard, adaptive bool) {
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
<td`)
	b.WriteString(classAttrIf(adaptive, "gsx-panel"))
	b.WriteString(` style="padding:12px 16px; background-color:`)
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
// technique: a td with font-size:0,
// line-height:0, and mso-line-height-rule:exactly pins an exact-height
// rule across every client; a plain border-top div does not. Parity mode
// keeps the original bare border-top div, byte-for-byte (and needs no
// class hook, for the same reason writeNote's parity branch does not).
// Hardened mode's rule gets gsx-border.
func writeDivider(b *strings.Builder, theme Theme, _ doc.ResolvedDivider, hard, adaptive bool) {
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
<tr><td`)
	b.WriteString(classAttrIf(adaptive, "gsx-border"))
	b.WriteString(` style="border-top:1px solid `)
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
// emailkit's own MarkRow semantics.
//
// Hardened mode applies the data-table contract: a StatTable holds
// facts, so it is a real table, never role="presentation", and its
// header cells are <th scope="col">, not <td>. Parity mode keeps the
// original role="presentation" and <td> header cells, byte-for-byte.
//
// adaptive marks the title (gsx-muted), the outer table (gsx-border), the
// header row's border and muted text (gsx-border, gsx-muted), and each
// data row's border plus its own color — gsx-copy normally, gsx-accent on
// the one marked row, alongside gsx-panel for that row's own background.
func writeStatTable(b *strings.Builder, theme Theme, s doc.ResolvedStatTable, hard, adaptive bool) {
	b.WriteString(`<tr>
<td style="padding:24px 32px 0 32px;">
`)
	if s.Title != "" {
		b.WriteString(`<div`)
		b.WriteString(classAttrIf(adaptive, "gsx-muted"))
		b.WriteString(` style="color:`)
		b.WriteString(theme.ColorMuted)
		b.WriteString(`; font-family:`)
		b.WriteString(theme.FontMono)
		b.WriteString(`; font-size:11px; letter-spacing:0.08em; text-transform:uppercase; margin-bottom:12px;">`)
		b.WriteString(escapeText(s.Title))
		b.WriteString(`</div>
`)
	}
	if hard {
		b.WriteString(`<table width="100%" cellpadding="0" cellspacing="0" border="0"`)
	} else {
		b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"`)
	}
	b.WriteString(classAttrIf(adaptive, "gsx-border"))
	b.WriteString(` style="width:100%; border:1px solid `)
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
			b.WriteString(classAttrIf2(adaptive, "gsx-border", "gsx-muted"))
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
		rowColorClass := "gsx-copy"
		background := ""
		rowClass := ""
		if i+1 == s.MarkRow {
			rowColor = theme.ColorAccent
			rowColorClass = "gsx-accent"
			background = ` background-color:` + theme.ColorPanel + `;`
			rowClass = classAttrIf(adaptive, "gsx-panel")
		}
		border := ""
		borderClass := ""
		if i < len(s.Rows)-1 {
			border = ` border-bottom:1px solid ` + theme.ColorBorder + `;`
			borderClass = "gsx-border"
		}
		b.WriteString(`<tr`)
		b.WriteString(rowClass)
		b.WriteString(` style="`)
		b.WriteString(background)
		b.WriteString("\">\n")
		for _, cell := range row.Cells {
			b.WriteString(`<td`)
			b.WriteString(classAttrIf2(adaptive, rowColorClass, borderClass))
			b.WriteString(` style="padding:10px 14px;`)
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

// writeCustomNode writes one Custom subtree node: a raw element with its
// literal (already lint-checked) attributes, or a
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

// writeFooter writes email.Footer. adaptive marks the border-top rule
// (gsx-border), the signoff (gsx-copy), and the fine-print note
// (gsx-faint).
func writeFooter(b *strings.Builder, theme Theme, f doc.ResolvedFooter, adaptive bool) {
	b.WriteString(`<tr>
<td style="padding:28px 32px 32px 32px;">
<div`)
	b.WriteString(classAttrIf2(adaptive, "gsx-border", "gsx-copy"))
	b.WriteString(` style="border-top:1px solid `)
	b.WriteString(theme.ColorBorder)
	b.WriteString(`; padding-top:20px; color:`)
	b.WriteString(theme.ColorBody)
	b.WriteString(`; font-family:`)
	b.WriteString(theme.FontSans)
	b.WriteString(`; font-size:14px;">`)
	b.WriteString(escapeText(f.Signoff))
	b.WriteString(`</div>
<div`)
	b.WriteString(classAttrIf(adaptive, "gsx-faint"))
	b.WriteString(` style="color:`)
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
