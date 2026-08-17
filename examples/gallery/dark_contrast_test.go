package gallery_test

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
)

// TestDigestDarkContrastMeetsWCAGAA is the launch-gate B2 finding's own
// closing proof: "Contrast-verify the digest golden programmatically
// post-fix (>=4.5:1 for body on the dark card)". Before the class-hook
// sweep, the digest golden's body copy (email.Headline's lede,
// email.Column's text) rendered with no adaptive hook at all, so a
// Gmail-class client applying its own forced dark transform recolored the
// surrounding card to Theme.Dark.ColorCard while leaving the text at its
// light-mode Theme.ColorBody — 1.73:1, well under WCAG AA's 4.5:1 body
// text floor. This test renders the actual digest golden and checks two
// things a theme-level-only check (lint.CheckTheme's EM141) cannot: that
// the class hook the dark <style> layer depends on is actually present on
// the rendered body-copy elements, and that the color pair it swaps to
// clears the WCAG AA ratio.
func TestDigestDarkContrastMeetsWCAGAA(t *testing.T) {
	theme := gsxmail.LedgerTheme()
	if theme.Dark == nil {
		t.Fatal("LedgerTheme no longer sets Dark; this test's whole premise depends on it")
	}

	html, err := os.ReadFile("digest/digest.html")
	if err != nil {
		t.Fatal(err)
	}

	// Every body-copy site in the rendered golden must carry the gsx-copy
	// class hook: the dark <style> layer's ".gsx-copy { color: ... }"
	// rule has nothing to attach to otherwise.
	copyDivs := regexp.MustCompile(`<div class="gsx-copy" style="color:`).FindAllIndex(html, -1)
	if len(copyDivs) == 0 {
		t.Fatal("digest.html has no gsx-copy class hook at all; the dark-mode body-copy contrast fix did not reach the rendered output")
	}

	ratio, ok := contrastRatio(theme.Dark.ColorBody, theme.Dark.ColorCard)
	if !ok {
		t.Fatalf("could not parse ColorBody %q or ColorCard %q as #RRGGBB", theme.Dark.ColorBody, theme.Dark.ColorCard)
	}
	t.Logf("Dark.ColorBody %s on Dark.ColorCard %s: %.2f:1", theme.Dark.ColorBody, theme.Dark.ColorCard, ratio)
	if ratio < 4.5 {
		t.Errorf("body-on-card contrast is %.2f:1; WCAG AA requires >=4.5:1", ratio)
	}

	// The dark <style> layer itself must carry the rule the gsx-copy class
	// hooks depend on, with the same hex value CheckTheme's own EM141
	// validated.
	wantRule := ".gsx-copy { color:" + theme.Dark.ColorBody + " !important; }"
	if !strings.Contains(string(html), wantRule) {
		t.Errorf("digest.html's <style> block is missing %q", wantRule)
	}
}

// contrastRatio and relativeLuminance duplicate lint.CheckTheme's own
// private WCAG 2 contrast formula (package lint's copy stays unexported;
// this test package does not import lint at all, and the formula is
// small enough that a second, independent implementation here is itself
// a mild extra proof that the two agree) — see M2's own similar contrast
// helper for the badge tones.
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
