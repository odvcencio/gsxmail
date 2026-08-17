package renderhtml

import (
	"math"
	"strconv"
	"strings"
)

// cardIsDark reports whether cardHex (a "#RRGGBB" color, ordinarily
// Theme.ColorCard) reads as a dark card rather than a light one: its own
// relative luminance is below the midpoint between white's (1.0) and
// #101611's (Terminal's own dark card, ~0.007) — badgeToneColor's own
// light/dark variant selector (M2, launch-gate findings). An unparseable
// cardHex reports false (the light-card variant), matching the shipped
// default theme's own white card.
func cardIsDark(cardHex string) bool {
	lum, ok := relativeLuminance(cardHex)
	if !ok {
		return false
	}
	return lum < 0.5
}

// relativeLuminance and contrastRatio duplicate lint.CheckTheme's own
// private WCAG 2 formula (that package's copy stays unexported, and
// renderhtml must not import lint — the render path never imports the
// check-time catalog, structural_isolation_test.go's own proof). Package
// gsxmail's examples/gallery/dark_contrast_test.go carries a third,
// independent copy for the same reason: the formula is small enough that
// a handful of independent copies is a fair trade against a shared
// dependency that would blur that isolation boundary.
func relativeLuminance(hex string) (float64, bool) {
	r, g, b, ok := hexToRGB(hex)
	if !ok {
		return 0, false
	}
	return 0.2126*srgbChannel(r) + 0.7152*srgbChannel(g) + 0.0722*srgbChannel(b), true
}

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
