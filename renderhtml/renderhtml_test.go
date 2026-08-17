package renderhtml

import "testing"

// TestBadgeToneColorClearsWCAGAA is M2's own closing proof (launch-gate
// findings): every status tone's light-card and dark-card variant must
// clear 4.5:1 (WCAG AA body text) against the card badgeToneColor
// actually selects it for — #FFFFFF (a light card, cardIsDark false) and
// #101611 (Terminal's own dark card, cardIsDark true) — computed
// programmatically here, not eyeballed once and pinned as a golden.
func TestBadgeToneColorClearsWCAGAA(t *testing.T) {
	lightCard := Theme{ColorCard: "#FFFFFF"}
	darkCard := Theme{ColorCard: "#101611"}
	if cardIsDark(lightCard.ColorCard) {
		t.Fatal("sanity: #FFFFFF must not read as a dark card")
	}
	if !cardIsDark(darkCard.ColorCard) {
		t.Fatal("sanity: #101611 must read as a dark card")
	}

	for _, tone := range []string{"positive", "warning", "critical"} {
		t.Run(tone, func(t *testing.T) {
			light := badgeToneColor(lightCard, tone)
			ratio, ok := contrastRatio(light, lightCard.ColorCard)
			if !ok {
				t.Fatalf("could not compute contrast for %q on %q", light, lightCard.ColorCard)
			}
			t.Logf("%s light-card variant %s on %s: %.2f:1", tone, light, lightCard.ColorCard, ratio)
			if ratio < 4.5 {
				t.Errorf("%s light-card variant %s on %s is %.2f:1; WCAG AA requires >=4.5:1", tone, light, lightCard.ColorCard, ratio)
			}

			dark := badgeToneColor(darkCard, tone)
			ratio, ok = contrastRatio(dark, darkCard.ColorCard)
			if !ok {
				t.Fatalf("could not compute contrast for %q on %q", dark, darkCard.ColorCard)
			}
			t.Logf("%s dark-card variant %s on %s: %.2f:1", tone, dark, darkCard.ColorCard, ratio)
			if ratio < 4.5 {
				t.Errorf("%s dark-card variant %s on %s is %.2f:1; WCAG AA requires >=4.5:1", tone, dark, darkCard.ColorCard, ratio)
			}
		})
	}
}

// TestLinkButtonSpacerWidth pins linkButtonSpacerWidth's own formula
// (launch-gate M1): half of (widthPx - estimatedTextWidth), floored at
// 0, where estimatedTextWidth reuses linkGlyphWidthPx per rune — the
// same per-rune estimate linkButtonDefaultWidth uses for the whole
// button's own default width.
func TestLinkButtonSpacerWidth(t *testing.T) {
	cases := []struct {
		name    string
		widthPx int
		label   string
		want    int
	}{
		{"wide-button-short-label", 220, "GO", 101},        // (220 - 2*9)/2 = 101
		{"exact-fit-label", 171, "sentinel-link-label", 0}, // 19 runes * 9 = 171; (171-171)/2 = 0
		{"label-wider-than-button-floors-at-0", 50, "a much longer label", 0},
		{"empty-label", 120, "", 60}, // (120 - 0)/2 = 60
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := linkButtonSpacerWidth(tc.widthPx, tc.label)
			if got != tc.want {
				t.Errorf("linkButtonSpacerWidth(%d, %q) = %d, want %d", tc.widthPx, tc.label, got, tc.want)
			}
			if got < 0 {
				t.Errorf("linkButtonSpacerWidth(%d, %q) = %d, must never be negative", tc.widthPx, tc.label, got)
			}
		})
	}
}
