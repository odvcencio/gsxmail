package gsxmail_test

import (
	"strings"
	"testing"

	"m31labs.dev/gsxmail/internal/doc"
	"m31labs.dev/gsxmail/renderhtml"
	"m31labs.dev/gsxmail/rendertext"
)

// TestURLSafetyWriterParity renders ONE resolved tree — carrying every
// URL site both writers judge, each with an unsafe value and a safe
// control — through both output contracts and proves the checkpoint's
// four claims hold across the parts simultaneously:
//
//  1. EM111 omission: no img src failing doc.SafeImgSrc reaches either
//     part, while every safe control still renders its <img>.
//  2. Alt fallback: each dropped image's alt text stands in — a styled
//     body-copy div in HTML, the same "[alt]" bracket form the text twin
//     has always used.
//  3. Findings: the writer's RenderFindings correspond one-to-one with
//     doc.AuditURLs' sites (EM110 per failing href, EM111 per failing img
//     src) — the audit oracle and the writer enforce exactly the same
//     set, so gsxmail.Set.Render's copy loop surfaces every drop in
//     Parts.Diagnostics.
//  4. EM110 parity: an href failing doc.SafeURL loses the link in HTML
//     and the ": URL"/"(url)" suffix in text — the text suffix appears
//     exactly when the HTML href does, for every href site (CTA, Button,
//     Custom <a>) and both directions (safe present, unsafe absent).
//
// Every value is a literal; both writers are pure functions of the
// resolved tree and theme, so every assertion is deterministic.
func TestURLSafetyWriterParity(t *testing.T) {
	const (
		badHeroSrc   = "http://cdn.example/unsafe-hero.png"
		badColumnSrc = "assets/column-relative.png"
		badCustomSrc = "//cdn.example/unsafe-custom.png"
		okHeroSrc    = "https://cdn.example/safe-hero.png"
		okColumnSrc  = "https://cdn.example/safe-column.png"
		okCustomSrc  = "https://cdn.example/safe-custom.png"

		badCTAHref    = "javascript:alert(document.cookie)"
		badBtnHref    = "data:text/html,x"
		badCustomHref = "ftp://files.example.com/pub"
		okCTAHref     = "https://example.com/claim"
		okMailtoHref  = "mailto:hi@example.com"
		okCustomHref  = "https://example.com/pub"

		heroAlt   = "hero stand-in"
		columnAlt = "column stand-in"
		customAlt = "custom stand-in"
	)

	resolved := &doc.Resolved{
		Shell: doc.ResolvedShell{Title: "url-safety parity", Lang: "en"},
		Blocks: []doc.ResolvedBlock{
			doc.ResolvedHero{Src: badHeroSrc, Alt: heroAlt, Width: "598", Height: "240"},
			doc.ResolvedHero{Src: okHeroSrc, Alt: "safe hero", Width: "598", Height: "240"},
			doc.ResolvedColumns{Columns: []doc.ResolvedColumn{
				{ImgSrc: badColumnSrc, ImgAlt: columnAlt, Title: "Unsafe column"},
				{ImgSrc: okColumnSrc, ImgAlt: "safe column", Title: "Safe column"},
			}},
			doc.ResolvedCTA{Label: "Claim seat", Href: badCTAHref},
			doc.ResolvedCTA{Label: "Visit", Href: okCTAHref},
			doc.ResolvedButton{Variant: "link", Label: "Data sheet", Href: badBtnHref},
			doc.ResolvedButton{Variant: "link", Label: "Mail us", Href: okMailtoHref},
			doc.ResolvedCustom{Root: doc.ResolvedCustomNode{
				Tag: "div",
				Children: []doc.ResolvedCustomNode{
					{Tag: "img",
						Attrs: []doc.ResolvedCustomAttr{{Name: "src", Value: badCustomSrc}, {Name: "alt", Value: customAlt}}},
					{Tag: "img",
						Attrs: []doc.ResolvedCustomAttr{{Name: "src", Value: okCustomSrc}, {Name: "alt", Value: "safe custom"}}},
					{Tag: "a",
						Attrs:    []doc.ResolvedCustomAttr{{Name: "href", Value: badCustomHref}},
						Children: []doc.ResolvedCustomNode{{IsText: true, Text: "Download files"}}},
					{Tag: "a",
						Attrs:    []doc.ResolvedCustomAttr{{Name: "href", Value: okCustomHref}},
						Children: []doc.ResolvedCustomNode{{IsText: true, Text: "Visit site"}}},
				},
			}},
		},
	}

	html, findings := renderhtml.WriteWithOptions(resolved, renderhtml.DefaultTheme(), renderhtml.WriteOptions{})
	text := rendertext.Write(resolved)
	sites := doc.AuditURLs(resolved)

	// --- Claim 3: findings ↔ audit oracle, one-to-one. ---
	if len(findings) != len(sites) {
		t.Fatalf("writer produced %d findings for %d audited sites; the oracle and the writer disagree:\nfindings: %+v\nsites: %+v",
			len(findings), len(sites), findings, sites)
	}
	for _, site := range sites {
		code := "EM110"
		if site.Field == "img src" {
			code = "EM111"
		}
		matched := false
		for _, f := range findings {
			if f.Code == code && strings.Contains(f.Message, site.Component) && strings.Contains(f.Message, site.Value) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("no %s finding matches audited site %+v;\nfindings: %+v", code, site, findings)
		}
	}

	// --- Claim 1: EM111 omission, both parts. ---
	for _, unsafe := range []string{badHeroSrc, badColumnSrc, badCustomSrc} {
		if strings.Contains(html, unsafe) {
			t.Errorf("HTML part leaks the rejected img src %q", unsafe)
		}
		if strings.Contains(text, unsafe) {
			t.Errorf("text part leaks the rejected img src %q", unsafe)
		}
	}
	for _, safe := range []string{okHeroSrc, okColumnSrc, okCustomSrc} {
		if !strings.Contains(html, safe) {
			t.Errorf("safe img src %q vanished from the HTML part; the check must not touch safe values", safe)
		}
	}

	// --- Claim 2: alt fallback, both parts. ---
	for _, pair := range []struct{ alt string }{{heroAlt}, {columnAlt}, {customAlt}} {
		if !strings.Contains(html, pair.alt) {
			t.Errorf("HTML part lost the alt stand-in %q for a dropped image", pair.alt)
		}
		if !strings.Contains(text, "["+pair.alt+"]") {
			t.Errorf("text part lost the alt stand-in [%q] for a dropped image", pair.alt)
		}
	}

	// --- Claim 4: EM110 parity — text suffix iff HTML href, per site. ---
	hrefSites := []struct {
		value string
		want  bool // whether the href should survive in both parts
	}{
		{badCTAHref, false},
		{badBtnHref, false},
		{badCustomHref, false},
		{okCTAHref, true},
		{okMailtoHref, true},
		{okCustomHref, true},
	}
	for _, s := range hrefSites {
		inHTML := strings.Contains(html, `href="`+s.value+`"`)
		inText := strings.Contains(text, ": "+s.value) || strings.Contains(text, "("+s.value+")")
		if inHTML != s.want {
			t.Errorf("HTML part: href %q presence = %v, want %v", s.value, inHTML, s.want)
		}
		if inText != s.want {
			t.Errorf("text part: URL %q suffix presence = %v, want %v", s.value, inText, s.want)
		}
		if inHTML != inText {
			t.Errorf("parts disagree on href %q: HTML=%v text=%v; the text suffix must appear exactly when the HTML href does", s.value, inHTML, inText)
		}
	}

	// Labels survive every rejected link, visibly, in both parts.
	for _, label := range []string{"Claim seat", "Data sheet", "Download files"} {
		if !strings.Contains(html, label) {
			t.Errorf("HTML part lost label %q along with its rejected href; only the link should drop", label)
		}
		if !strings.Contains(text, label) {
			t.Errorf("text part lost label %q along with its rejected URL; only the suffix should drop", label)
		}
	}
}
