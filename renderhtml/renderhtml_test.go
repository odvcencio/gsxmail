package renderhtml

import (
	"regexp"
	"strings"
	"testing"

	"m31labs.dev/gsxmail/internal/doc"
)

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

// imgTagPattern finds every emitted <img ...> tag, hero and column images
// alike (issue #3's own two sites — writeHero and writeColumnContent).
var imgTagPattern = regexp.MustCompile(`<img\b[^>]*>`)

// heroColumnFixture is one Resolved doc carrying both of issue #3's
// img-emitting sites: a Hero and a Columns block with one Column image.
func heroColumnFixture() *doc.Resolved {
	return &doc.Resolved{
		Shell: doc.ResolvedShell{Title: "issue-3 fixture", Lang: "en"},
		Blocks: []doc.ResolvedBlock{
			doc.ResolvedHero{
				Src: "https://example.com/hero@2x.png", Alt: "hero alt text",
				Width: "598", Height: "240",
			},
			doc.ResolvedColumns{Columns: []doc.ResolvedColumn{{
				ImgSrc: "https://example.com/col@2x.png", ImgAlt: "column alt text",
				ImgWidth: "120", ImgHeight: "80",
			}}},
		},
	}
}

// TestImgAltTextIsStyled is issue #3's own structural proof: every emitted
// <img> — Hero's and Column's, gsxmail's only two synthesized img sites
// (Custom's raw <img> pass-through is deliberately out of scope, per
// writeCustomNode's own doc comment) — carries font-family, font-size,
// and color in its style attribute, so a blocked or missing image's alt
// text renders in the template's own font and ink color instead of a
// mail client's default (typically blue, serif) alt-text style.
func TestImgAltTextIsStyled(t *testing.T) {
	for _, hard := range []bool{true, false} {
		html, _ := WriteWithOptions(heroColumnFixture(), DefaultTheme(), WriteOptions{Outlook: map[bool]string{true: "", false: "off"}[hard]})
		imgs := imgTagPattern.FindAllString(html, -1)
		if len(imgs) != 2 {
			t.Fatalf("hard=%v: got %d <img> tags, want 2 (one Hero, one Column)", hard, len(imgs))
		}
		for _, img := range imgs {
			for _, want := range []string{"font-family:", "font-size:", "color:"} {
				if !strings.Contains(img, want) {
					t.Errorf("hard=%v: <img> tag missing %q in its style:\n%s", hard, want, img)
				}
			}
		}
	}
}

// TestImgAltTextDarkModeHook proves the adaptive dark-mode class hook
// (gsx-copy, the same token every other body-copy element carries —
// writeDarkStyleLayer's own doc comment) reaches an img's alt-text color
// exactly when the active theme's own DarkStrategy is "adaptive", and
// never otherwise — DefaultTheme's "none" strategy must render byte-
// identically to before this fix on that axis: no class attribute at all.
func TestImgAltTextDarkModeHook(t *testing.T) {
	adaptiveTheme := LedgerTheme() // DarkMode "adaptive" (theme.go)
	html, _ := Write(heroColumnFixture(), adaptiveTheme)
	imgs := imgTagPattern.FindAllString(html, -1)
	if len(imgs) != 2 {
		t.Fatalf("got %d <img> tags, want 2", len(imgs))
	}
	for _, img := range imgs {
		if !strings.Contains(img, `class="gsx-copy"`) {
			t.Errorf("adaptive theme: <img> tag missing the gsx-copy dark-mode hook:\n%s", img)
		}
	}

	noneTheme := DefaultTheme() // DarkMode "none"
	html, _ = Write(heroColumnFixture(), noneTheme)
	imgs = imgTagPattern.FindAllString(html, -1)
	if len(imgs) != 2 {
		t.Fatalf("got %d <img> tags, want 2", len(imgs))
	}
	for _, img := range imgs {
		if strings.Contains(img, "class=") {
			t.Errorf("DarkMode \"none\" theme: <img> tag must carry no class attribute:\n%s", img)
		}
	}
}
