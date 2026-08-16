package importer

import (
	"strconv"
	"strings"
)

// shellInfo is everything Import recovers about the template's outer
// frame: the ghost-table/card shell that maps to <email.Shell> (pixel
// dossier section 7.2(1), heuristic 1), its preheader (heuristic 2), and
// the color/font tokens a generated Theme literal reproduces.
type shellInfo struct {
	title     string
	lang      string
	wordmark  string
	tagline   string
	shortCode string
	preheader string

	cardWidth int
	theme     themeTokens
}

// themeTokens is the palette Import extracts from the card shell's own
// literal colors (task instructions, design constraint 2: "theme-token
// extraction from the dominant colors into a generated Theme literal").
// Every field defaults to DefaultTheme()'s own value, so a source that
// carries no recognizable color for a given slot still produces a valid,
// readable Theme rather than an empty string.
type themeTokens struct {
	ColorGround, ColorCard, ColorPanel, ColorBorder string
	ColorAccent, ColorInk, ColorBody, ColorMuted    string
	ColorFaint                                      string
	FontSans, FontMono                              string
}

func defaultThemeTokens() themeTokens {
	return themeTokens{
		ColorGround: "#F4F4F6", ColorCard: "#FFFFFF", ColorPanel: "#F7F7F9",
		ColorBorder: "#E2E2E8", ColorAccent: "#3452FF", ColorInk: "#16161D",
		ColorBody: "#3C3C46", ColorMuted: "#71717F", ColorFaint: "#9C9CA8",
		FontSans: "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif",
		FontMono: "'SFMono-Regular',Consolas,Menlo,monospace",
	}
}

// findBody returns src's <body> element, or src itself when the source
// has no <html>/<body> wrapper at all (a bare content fragment — a valid
// gotreesitter parse either way, per the grammar's own tolerance).
func findBody(root *node) *node {
	if b := findFirst(root, "body"); b != nil {
		return b
	}
	return root
}

// findCard locates the card table heuristic 1 maps to <email.Shell>: the
// highest-scoring <table> whose own width (an HTML width attribute or a
// CSS width/max-width declaration) falls in an email card's plausible
// range. Scoring by row count first, then by inner text volume, prefers
// the outermost table that actually holds the message's content over an
// inner single-cell wrapper or a narrow nested table (a Panel's own
// bordered table, for instance) — MJML and react-email's compiled output
// both nest one or more purely structural nest tables around the real
// card (heuristic 1's own "ghost tables that duplicate an adjacent
// structure map to nothing"); scoring naturally prefers the row-richest
// candidate over a structural wrapper with just one row.
func findCard(body *node) *node {
	var best *node
	bestScore := -1
	for _, t := range findAll(body, "table") {
		w, ok := tableWidth(t)
		if !ok || w < 320 || w > 820 {
			continue
		}
		rows := 0
		for _, e := range t.elements() {
			if e.tag == "tr" {
				rows++
			}
		}
		if rows == 0 {
			continue
		}
		score := rows*1000 + len(t.innerText())
		if score > bestScore {
			bestScore = score
			best = t
		}
	}
	return best
}

// tableWidth reads a plausible pixel width for t from its "width"
// attribute or its style's width/max-width declaration, in that order.
func tableWidth(t *node) (int, bool) {
	if v, ok := t.attr("width"); ok {
		if n, ok := parsePixels(v); ok {
			return n, true
		}
	}
	for _, prop := range []string{"width", "max-width"} {
		if v, ok := t.styleValue(prop); ok {
			if n, ok := parsePixels(v); ok {
				return n, true
			}
		}
	}
	return 0, false
}

// parsePixels reads a leading decimal integer off s, tolerating a
// trailing "px" or "%" (a percent value is rejected: it never names a
// plausible fixed card width).
func parsePixels(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		return 0, false
	}
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// findPreheader returns body's own hidden inbox-preview div (heuristic
// 2): the first element child of body whose style declares
// "display:none" together with "overflow:hidden" or "opacity:0" — the
// react-email/gsxmail suppression-style cluster (pixel dossier section
// 6.1) — and its decoded, pad-stripped text. It returns "" when body's
// first element child does not match, rather than searching the whole
// tree: a preheader is defined by its position (first child of body), not
// just its style, so a later hidden div (a display:none utility class
// elsewhere in the document) is deliberately not read as one.
func findPreheader(body *node) string {
	first := firstElement(body)
	if first == nil || first.tag != "div" {
		return ""
	}
	decl := first.style()
	hasDisplayNone := false
	hasSuppression := false
	for _, d := range decl {
		if d.prop == "display" && strings.Contains(d.value, "none") {
			hasDisplayNone = true
		}
		if (d.prop == "overflow" && strings.Contains(d.value, "hidden")) ||
			(d.prop == "opacity" && strings.TrimSpace(d.value) == "0") {
			hasSuppression = true
		}
	}
	if !hasDisplayNone || !hasSuppression {
		return ""
	}
	return stripPreheaderPad(first.innerText())
}

// stripPreheaderPad removes writePreheader's own trailing
// &nbsp;/&zwnj;-pad tail (already decoded to U+00A0/U+200C by domify's
// entity unescaping), so props.sample.json carries the author's own
// preheader text, not gsxmail's own 150-character padding.
func stripPreheaderPad(s string) string {
	return strings.TrimRight(s, "\u00A0\u200C \t")
}

func firstElement(n *node) *node {
	for i := range n.children {
		c := &n.children[i]
		if !c.isText && !c.isComment {
			return c
		}
	}
	return nil
}

// findTitle reads <title>'s text, or "" when absent.
func findTitle(root *node) string {
	if t := findFirst(root, "title"); t != nil {
		return t.innerText()
	}
	return ""
}

// findLang reads <html lang="...">, defaulting to "en" — every shipped
// gsxmail template sets one, and EM-catalog has no rule requiring a
// generated template to as well, but "en" is a safe, honest default.
func findLang(root *node) string {
	if h := findFirst(root, "html"); h != nil {
		if v, ok := h.attr("lang"); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "en"
}
