package lint

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"m31labs.dev/gsxmail/renderhtml"
)

// CheckTheme runs the WP5.2 Theme-level dark-mode checks (design spec
// section 15, WP5.2; pixel dossier section 5, EM140/EM141/EM142/EM144)
// once per Load call, against the Set's own Options.Theme rather than any
// one template's source. gsxmail.Load appends its result to the same
// diagnostic list every template's own CheckProgram call contributes to,
// so an error-severity finding here fails Load closed exactly like any
// other (EM143, the one remaining WP5.2 theme-adjacent rule, is a
// per-template AST check instead — see catalog.go's checkDarkPaletteCoverage).
//
// A Diagnostic here carries no File/Line/Col, the same "no source
// position" shape gsxmail.SizeBudgetError's own Diagnostic uses: the
// theme, not template source, is what the finding is about.
func CheckTheme(theme renderhtml.Theme) []Diagnostic {
	var diags []Diagnostic
	add := func(code, severity, msg string) {
		diags = append(diags, Diagnostic{Code: code, Severity: severity, Message: msg})
	}

	strategy := theme.DarkStrategy()

	if strategy == "adaptive" && theme.Dark == nil {
		add("EM140", "error", `theme sets DarkMode "adaptive" but Theme.Dark is nil; adaptive mode requires a dark palette`)
	}

	// EM144: an explicit ColorScheme must agree with what DarkMode implies.
	// "none" is unconstrained — ColorScheme alone still drives the meta
	// tags, exactly as WP5.1 shipped. "locked" implies a dark-native meta,
	// so any ColorScheme other than "dark" is a real conflict. "adaptive"
	// computes "light dark" itself (renderhtml.writeHead overrides the
	// meta outright), so any single-value ColorScheme is stale or
	// mistaken.
	switch strategy {
	case "locked":
		if theme.ColorScheme != "" && theme.ColorScheme != "dark" {
			add("EM144", "error", fmt.Sprintf(`meta color-scheme %q conflicts with theme DarkMode %q`, theme.ColorScheme, theme.DarkMode))
		}
	case "adaptive":
		if theme.ColorScheme != "" {
			add("EM144", "error", fmt.Sprintf(`meta color-scheme %q conflicts with theme DarkMode %q`, theme.ColorScheme, theme.DarkMode))
		}
	}

	// EM141 (contrast) and EM142 (pure black/white) run only over the
	// color set that is actually shown as a dark presentation: "locked"
	// means Theme's own fields are dark-native, so they are checked
	// directly; "adaptive" means Theme.Dark is the swapped-in dark
	// presentation, so Theme's own (light) fields are deliberately left
	// unchecked — a light theme's white card is normal, not a dark-mode
	// design smell (pixel dossier section 5.3, rules 1-2). "none" runs
	// neither check: a plain light theme never participates in a dark
	// strategy at all.
	switch strategy {
	case "locked":
		checkPalette(&diags, "", themeColorMap(theme))
	case "adaptive":
		if theme.Dark != nil {
			checkPalette(&diags, "Dark.", darkColorMap(theme.Dark))
		}
	}

	return diags
}

func themeColorMap(t renderhtml.Theme) map[string]string {
	return map[string]string{
		"ColorGround": t.ColorGround, "ColorCard": t.ColorCard, "ColorPanel": t.ColorPanel,
		"ColorBorder": t.ColorBorder, "ColorAccent": t.ColorAccent, "ColorInk": t.ColorInk,
		"ColorBody": t.ColorBody, "ColorMuted": t.ColorMuted, "ColorFaint": t.ColorFaint,
	}
}

func darkColorMap(d *renderhtml.DarkPalette) map[string]string {
	return map[string]string{
		"ColorGround": d.ColorGround, "ColorCard": d.ColorCard, "ColorPanel": d.ColorPanel,
		"ColorBorder": d.ColorBorder, "ColorAccent": d.ColorAccent, "ColorInk": d.ColorInk,
		"ColorBody": d.ColorBody, "ColorMuted": d.ColorMuted, "ColorFaint": d.ColorFaint,
	}
}

// checkPalette runs EM142 (pure black/white) over every color in colors,
// and EM141 (WCAG AA body-text contrast, 4.5:1) over the two token pairs
// that actually carry body text: ColorInk-on-ColorCard and
// ColorBody-on-ColorCard (pixel dossier section 5.3, rule 2: "Every
// text/background token pair ... must clear 4.5:1"; scoped to the two text
// tokens against the card background they render on, rather than every
// possible pairing). labelPrefix distinguishes a light Theme's own tokens
// ("") from Theme.Dark's ("Dark.") in a message.
func checkPalette(diags *[]Diagnostic, labelPrefix string, colors map[string]string) {
	for _, field := range []string{"ColorGround", "ColorCard", "ColorPanel", "ColorBorder", "ColorAccent", "ColorInk", "ColorBody", "ColorMuted", "ColorFaint"} {
		hex := strings.ToUpper(colors[field])
		if hex == "#000000" || hex == "#FFFFFF" {
			*diags = append(*diags, Diagnostic{
				Code: "EM142", Severity: "warn",
				Message: fmt.Sprintf(`theme color %s is pure black or pure white; forced dark-mode transforms map extremes hard — use midtones`, labelPrefix+field),
			})
		}
	}

	for _, pair := range [][2]string{{"ColorInk", "ColorCard"}, {"ColorBody", "ColorCard"}} {
		fg, bg := pair[0], pair[1]
		ratio, ok := contrastRatio(colors[fg], colors[bg])
		if !ok {
			continue
		}
		if ratio < 4.5 {
			*diags = append(*diags, Diagnostic{
				Code: "EM141", Severity: "error",
				Message: fmt.Sprintf("theme contrast %s on %s is %.1f:1; body text requires 4.5:1 (WCAG AA)", labelPrefix+fg, labelPrefix+bg, ratio),
			})
		}
	}
}

// contrastRatio computes the WCAG 2 contrast ratio between two "#RRGGBB"
// colors: (L1+0.05)/(L2+0.05), where L1 is the lighter relative luminance
// (the standard formula EM141 checks against).
func contrastRatio(fgHex, bgHex string) (float64, bool) {
	l1, ok1 := relativeLuminance(fgHex)
	l2, ok2 := relativeLuminance(bgHex)
	if !ok1 || !ok2 {
		return 0, false
	}
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05), true
}

// relativeLuminance implements the WCAG 2 relative-luminance formula over
// an sRGB "#RRGGBB" color.
func relativeLuminance(hex string) (float64, bool) {
	r, g, b, ok := hexToRGB(hex)
	if !ok {
		return 0, false
	}
	return 0.2126*srgbChannel(r) + 0.7152*srgbChannel(g) + 0.0722*srgbChannel(b), true
}

func srgbChannel(c float64) float64 {
	c /= 255
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func hexToRGB(hex string) (r, g, b float64, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	r = float64((v >> 16) & 0xFF)
	g = float64((v >> 8) & 0xFF)
	b = float64(v & 0xFF)
	return r, g, b, true
}
