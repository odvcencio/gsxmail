package renderhtml

import (
	"strings"
	"testing"

	"m31labs.dev/gsxmail/internal/doc"
)

// Shared fixture values for this file's URL-safety proofs. Every value is
// a literal, so every assertion below is byte-deterministic.
const (
	unsafeHeroSrc   = "http://cdn.example/unsafe-hero.png"  // http: fails EM111
	unsafeColumnSrc = "assets/column-relative.png"          // relative path: fails EM111
	unsafeCustomSrc = "//cdn.example/unsafe-custom.png"     // protocol-relative: fails EM111
	safeHeroSrc     = "https://cdn.example/safe-hero.png"   // control
	safeColumnSrc   = "https://cdn.example/safe-column.png" // control
	safeCustomSrc   = "https://cdn.example/safe-custom.png" // control

	unsafeCTAHref    = "javascript:alert(document.cookie)" // fails doc.SafeURL
	unsafeButtonHref = "data:text/html,x"                  // fails doc.SafeURL
	unsafeLinkHref   = "vbscript:Execute"                  // fails doc.SafeURL
	unsafeCustomHref = "ftp://files.example.com/pub"       // fails doc.SafeURL
	safeCTAHref      = "https://example.com/claim"         // control
	safeMailtoHref   = "mailto:hi@example.com"             // control

	heroAltFallback   = "Hero stand-in text"
	columnAltFallback = "Column stand-in text"
	customAltFallback = "Custom stand-in text"
)

// urlSafetyFixture is one Resolved tree carrying every image site the two
// writers judge (Hero, Column, Custom <img>) plus every href site
// (CTA, Button secondary, Button link, Custom <a>) — each with one unsafe
// value and one safe control — so a single WriteWithOptions run exercises
// omission, fallback, and findings for all of them at once.
func urlSafetyFixture() *doc.Resolved {
	return &doc.Resolved{
		Shell: doc.ResolvedShell{Title: "url-safety fixture", Lang: "en"},
		Blocks: []doc.ResolvedBlock{
			doc.ResolvedHero{Src: unsafeHeroSrc, Alt: heroAltFallback, Width: "598", Height: "240"},
			doc.ResolvedHero{Src: safeHeroSrc, Alt: "safe hero", Width: "598", Height: "240"},
			doc.ResolvedColumns{Columns: []doc.ResolvedColumn{
				{ImgSrc: unsafeColumnSrc, ImgAlt: columnAltFallback, Title: "Unsafe column"},
				{ImgSrc: safeColumnSrc, ImgAlt: "safe column", Title: "Safe column"},
				{Title: "Image-free column"}, // no imgSrc: nothing to judge
			}},
			doc.ResolvedCTA{Label: "Act now", Href: unsafeCTAHref},
			doc.ResolvedButton{Variant: "secondary", Label: "Read docs", Href: unsafeButtonHref},
			doc.ResolvedButton{Variant: "link", Label: "Open guide", Href: unsafeLinkHref},
			doc.ResolvedButton{Variant: "link", Label: "Mail us", Href: safeMailtoHref},
			doc.ResolvedCustom{Root: doc.ResolvedCustomNode{
				Tag: "div",
				Children: []doc.ResolvedCustomNode{
					{Tag: "img",
						Attrs: []doc.ResolvedCustomAttr{{Name: "src", Value: unsafeCustomSrc}, {Name: "alt", Value: customAltFallback}}},
					{Tag: "img",
						Attrs: []doc.ResolvedCustomAttr{{Name: "src", Value: safeCustomSrc}, {Name: "alt", Value: "safe custom"}}},
					{Tag: "a",
						Attrs:    []doc.ResolvedCustomAttr{{Name: "href", Value: unsafeCustomHref}},
						Children: []doc.ResolvedCustomNode{{IsText: true, Text: "Download files"}}},
					{Tag: "a",
						Attrs:    []doc.ResolvedCustomAttr{{Name: "href", Value: safeCTAHref}},
						Children: []doc.ResolvedCustomNode{{IsText: true, Text: "Visit site"}}},
				},
			}},
		},
	}
}

// wantEM111Messages are the exact RenderFinding messages the fixture's
// three unsafe images must produce, byte for byte.
var wantEM111Messages = []string{
	`email.Hero img src "` + unsafeHeroSrc + `" is not an absolute https URL; rendering its alt text instead`,
	`email.Column img src "` + unsafeColumnSrc + `" is not an absolute https URL; rendering its alt text instead`,
	`email.Custom img src "` + unsafeCustomSrc + `" is not an absolute https URL; rendering its alt text instead`,
}

// wantEM110Messages are the exact RenderFinding messages the fixture's
// four unsafe hrefs must produce, byte for byte.
var wantEM110Messages = []string{
	`email.CTA href "` + unsafeCTAHref + `" uses a disallowed scheme (allowed: https, http, mailto); rendering the label without a link`,
	`email.Button href "` + unsafeButtonHref + `" uses a disallowed scheme (allowed: https, http, mailto); rendering the label without a link`,
	`email.Button href "` + unsafeLinkHref + `" uses a disallowed scheme (allowed: https, http, mailto); rendering the label without a link`,
	`email.Custom href "` + unsafeCustomHref + `" uses a disallowed scheme (allowed: https, http, mailto); rendering the label without a link`,
}

// TestEM111UnsafeImgSrcOmittedFromHTML is the EM111 omission proof: after
// WriteWithOptions over urlSafetyFixture, none of the three unsafe src
// values may appear anywhere in the HTML part, and the only <img> tags
// present are the three safe controls. An unsafe value must never reach
// the output even though nothing here ran Load-time lint — a caller
// driving the writer directly with props-resolved values is exactly the
// case the emission-time check exists for.
func TestEM111UnsafeImgSrcOmittedFromHTML(t *testing.T) {
	html, findings := WriteWithOptions(urlSafetyFixture(), DefaultTheme(), WriteOptions{})

	for _, unsafe := range []string{unsafeHeroSrc, unsafeColumnSrc, unsafeCustomSrc} {
		if strings.Contains(html, unsafe) {
			t.Errorf("EM111 omission failed: unsafe img src %q reached the HTML output:\n%s", unsafe, html)
		}
	}

	imgs := imgTagPattern.FindAllString(html, -1)
	if len(imgs) != 3 {
		t.Fatalf("got %d <img> tags, want exactly 3 (the safe Hero, Column, and Custom controls):\n%s",
			len(imgs), strings.Join(imgs, "\n"))
	}
	for _, safe := range []string{safeHeroSrc, safeColumnSrc, safeCustomSrc} {
		found := false
		for _, img := range imgs {
			if strings.Contains(img, safe) {
				found = true
			}
		}
		if !found {
			t.Errorf("control lost: safe img src %q is missing from the output; the safety check must not touch safe values", safe)
		}
	}
	if len(findings) != 7 { // 3 EM111 + 4 EM110, counted exactly in the findings tests below
		t.Fatalf("got %d findings, want 7; the omission must stay visible, not silent:\n%+v", len(findings), findings)
	}
}

// standInDivClass reports whether the <div> immediately wrapping alt in
// html carries the adaptive gsx-copy hook.
func standInDivClass(t *testing.T, html, alt string) bool {
	t.Helper()
	i := strings.Index(html, alt)
	if i < 0 {
		t.Fatalf("alt %q not found in output", alt)
	}
	start := strings.LastIndex(html[:i], "<div")
	if start < 0 {
		t.Fatalf("alt %q is not wrapped in a <div>", alt)
	}
	tagEnd := start + strings.Index(html[start:i], ">")
	return strings.Contains(html[start:tagEnd], `class="gsx-copy"`)
}

// TestEM111AltFallbackRenders is the alt-fallback proof: where an unsafe
// image is dropped, its alt text stands in. Hero and Column render their
// alt in a styled body-copy <div> ("line-height:1.5;" / the Column's own
// "margin-bottom:12px;" tail pins the styled-div shape byte-exactly);
// a Custom <img>'s alt renders as the bare escaped text run writeCustomNode
// emits. Under DefaultTheme (dark strategy "none") neither stdlib stand-in
// carries a class attribute; under LedgerTheme (adaptive) both carry the
// same gsx-copy dark-mode hook every other body-copy element does.
func TestEM111AltFallbackRenders(t *testing.T) {
	html, _ := WriteWithOptions(urlSafetyFixture(), DefaultTheme(), WriteOptions{})
	for _, want := range []string{
		`line-height:1.5;">` + heroAltFallback + `</div>`,      // Hero stand-in: styled div
		`margin-bottom:12px;">` + columnAltFallback + `</div>`, // Column stand-in: styled div
		">" + customAltFallback + "<",                          // Custom stand-in: bare text run
	} {
		if !strings.Contains(html, want) {
			t.Errorf("alt fallback missing %q from the HTML output:\n%s", want, html)
		}
	}
	if standInDivClass(t, html, heroAltFallback) || standInDivClass(t, html, columnAltFallback) {
		t.Error("DefaultTheme (strategy \"none\") must add no class attribute to either stand-in div")
	}

	adaptive, _ := WriteWithOptions(urlSafetyFixture(), LedgerTheme(), WriteOptions{})
	if !standInDivClass(t, adaptive, heroAltFallback) {
		t.Error("adaptive theme: Hero stand-in div is missing the gsx-copy dark-mode hook")
	}
	if !standInDivClass(t, adaptive, columnAltFallback) {
		t.Error("adaptive theme: Column stand-in div is missing the gsx-copy dark-mode hook")
	}
}

// TestEM111RenderFindingsAreVisible is the findings proof: each dropped
// image produces exactly one EM111 RenderFinding naming the component and
// the rejected src, so Set.Render can surface the drop in
// Parts.Diagnostics instead of silently swallowing it — and an all-safe
// tree produces zero findings of any code.
func TestEM111RenderFindingsAreVisible(t *testing.T) {
	_, findings := WriteWithOptions(urlSafetyFixture(), DefaultTheme(), WriteOptions{})

	var em111 int
	for _, want := range wantEM111Messages {
		count := 0
		for _, f := range findings {
			if f.Code == "EM111" && f.Message == want {
				count++
			}
		}
		if count != 1 {
			t.Errorf("want exactly one EM111 finding %q, got %d;\nfindings: %+v", want, count, findings)
		}
		em111 += count
	}
	if em111 != 3 {
		t.Errorf("got %d EM111 findings total, want exactly 3 (one per dropped image)", em111)
	}

	// All-safe control: zero findings of any kind.
	safe := &doc.Resolved{Blocks: []doc.ResolvedBlock{
		doc.ResolvedHero{Src: safeHeroSrc, Alt: "safe"},
		doc.ResolvedColumns{Columns: []doc.ResolvedColumn{
			{ImgSrc: safeColumnSrc, ImgAlt: "safe"},
			{Title: "image-free"},
		}},
	}}
	if _, f := WriteWithOptions(safe, DefaultTheme(), WriteOptions{}); len(f) != 0 {
		t.Errorf("all-safe tree produced %d findings, want 0:\n%+v", len(f), f)
	}
}

// TestAppendImgSrcRejectedNilContract pins appendImgSrcRejected's nil
// contract: appending to a nil *[]RenderFinding is a deliberate no-op,
// never a panic (the same guarantee appendHrefRejected documents).
func TestAppendImgSrcRejectedNilContract(t *testing.T) {
	appendImgSrcRejected(nil, "email.Hero", unsafeHeroSrc)
	appendHrefRejected(nil, "email.CTA", unsafeCTAHref)
}

// TestEM110ParityHrefSitesDroppedInHTML is the HTML side of the EM110
// parity proof: every href site the writer judges — CTA, Button
// secondary, Button link, and a Custom-subtree <a> — drops its link when
// doc.SafeURL rejects the scheme, keeps rendering its label, and records
// an EM110 finding naming the component; every safe control (https and
// mailto) keeps its clickable href untouched. The text twin of this claim
// lives in rendertext's own urlsafe_test.go and the cross-writer version
// in ../urlsafe_parity_test.go.
func TestEM110ParityHrefSitesDroppedInHTML(t *testing.T) {
	html, findings := WriteWithOptions(urlSafetyFixture(), DefaultTheme(), WriteOptions{})

	for _, unsafe := range []string{unsafeCTAHref, unsafeButtonHref, unsafeLinkHref, unsafeCustomHref} {
		if strings.Contains(html, unsafe) {
			t.Errorf("EM110 omission failed: unsafe href %q reached the HTML output", unsafe)
		}
	}
	if strings.Contains(html, `<a href="`+unsafeCTAHref+`"`) ||
		strings.Contains(html, `<a href="`+unsafeButtonHref+`"`) ||
		strings.Contains(html, `<a href="`+unsafeLinkHref+`"`) ||
		strings.Contains(html, `<a href="`+unsafeCustomHref+`"`) {
		t.Error("an unsafe href still renders as a clickable <a>; the link must drop")
	}

	// Labels survive the drop, visibly.
	for _, label := range []string{"Act now", "Read docs", "Open guide", "Download files"} {
		if !strings.Contains(html, label) {
			t.Errorf("label %q vanished along with its rejected href; only the link should drop", label)
		}
	}

	// Safe controls stay clickable.
	for _, want := range []string{`<a href="` + safeMailtoHref + `"`, `<a href="` + safeCTAHref + `"`} {
		if !strings.Contains(html, want) {
			t.Errorf("safe control href missing from output: %q", want)
		}
	}

	// Exactly the four expected EM110 findings, byte for byte.
	var em110 int
	for _, want := range wantEM110Messages {
		count := 0
		for _, f := range findings {
			if f.Code == "EM110" && f.Message == want {
				count++
			}
		}
		if count != 1 {
			t.Errorf("want exactly one EM110 finding %q, got %d", want, count)
		}
		em110 += count
	}
	if em110 != 4 {
		t.Errorf("got %d EM110 findings total, want exactly 4 (one per rejected href)", em110)
	}
}

// TestWriterFindingsMatchAuditURLsOracle proves AuditURLs' coverage-oracle
// claim for this writer: the findings WriteWithOptions returns over a tree
// correspond one-to-one with the sites AuditURLs reports — same count,
// same components, same values, EM110 for every failing href and EM111
// for every failing img src. If the audit ever disagrees with the writer,
// one of them enforces less than the other and this test names it.
func TestWriterFindingsMatchAuditURLsOracle(t *testing.T) {
	sites := doc.AuditURLs(urlSafetyFixture())
	_, findings := WriteWithOptions(urlSafetyFixture(), DefaultTheme(), WriteOptions{})

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
}
