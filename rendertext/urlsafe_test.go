package rendertext

import (
	"strings"
	"testing"

	"m31labs.dev/gsxmail/internal/doc"
)

// Shared fixture values for this file's URL-safety proofs. Every value is
// a literal, so every assertion below is byte-deterministic.
const (
	unsafeHeroSrc   = "http://cdn.example/unsafe-hero.png" // fails doc.SafeImgSrc
	unsafeColumnSrc = "assets/column-relative.png"         // fails doc.SafeImgSrc
	unsafeCustomSrc = "//cdn.example/unsafe-custom.png"    // fails doc.SafeImgSrc

	unsafeCTAHref   = "javascript:alert(document.cookie)" // fails doc.SafeURL
	unsafeButtonSrc = "data:text/html,x"                  // fails doc.SafeURL
	unsafeLinkHref  = "vbscript:Execute"                  // fails doc.SafeURL

	heroAltFallback   = "Hero stand-in text"
	columnAltFallback = "Column stand-in text"
	customAltFallback = "Custom stand-in text"
)

// TestEM110ParityTextHrefSuffix is the text side of the EM110 parity
// proof: an href failing doc.SafeURL renders "-> LABEL" with NO ": URL"
// suffix on every block that carries one — email.CTA and both non-primary
// Button variants (primary delegates to CTA byte-identically) — while the
// same labels with https and mailto hrefs keep their suffixes. The exact
// full-text equality below pins the whole shape at once: nothing but the
// suffix may differ between the rejected and accepted cases, which is
// exactly the parity the HTML side enforces by dropping only the href
// attribute. The cross-writer version lives in ../urlsafe_parity_test.go.
func TestEM110ParityTextHrefSuffix(t *testing.T) {
	rejected := &doc.Resolved{Blocks: []doc.ResolvedBlock{
		doc.ResolvedCTA{Label: "Claim seat", Href: unsafeCTAHref},
		doc.ResolvedButton{Variant: "secondary", Label: "Read docs", Href: unsafeButtonSrc},
		doc.ResolvedButton{Variant: "link", Label: "Open guide", Href: unsafeLinkHref},
	}}
	want := " // \n\n  -> Claim seat\n\n  -> Read docs\n\n  -> Open guide"
	if got := Write(rejected); got != want {
		t.Errorf("rejected-href text mismatch.\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}

	accepted := &doc.Resolved{Blocks: []doc.ResolvedBlock{
		doc.ResolvedCTA{Label: "Claim seat", Href: "https://example.com/claim"},
		doc.ResolvedButton{Variant: "secondary", Label: "Read docs", Href: "https://example.com/docs"},
		doc.ResolvedButton{Variant: "link", Label: "Open guide", Href: "mailto:hi@example.com"},
	}}
	want = " // \n\n  -> Claim seat: https://example.com/claim\n\n  -> Read docs: https://example.com/docs\n\n  -> Open guide: mailto:hi@example.com"
	if got := Write(accepted); got != want {
		t.Errorf("accepted-href text mismatch.\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

// TestEM110ParityTextCustomLinkRendersLabelOnly proves the Custom-subtree
// <a> gets the same fail-closed judgment in text as in HTML: an href
// failing doc.SafeURL leaves only the label ("Download files"), while the
// safe control keeps its "label (url)" shape — deriveLinkText drops the
// URL suffix exactly where writeCustomNode drops the href attribute.
func TestEM110ParityTextCustomLinkRendersLabelOnly(t *testing.T) {
	resolved := &doc.Resolved{Blocks: []doc.ResolvedBlock{
		doc.ResolvedCustom{Root: doc.ResolvedCustomNode{
			Tag: "div",
			Children: []doc.ResolvedCustomNode{
				{Tag: "p", Children: []doc.ResolvedCustomNode{
					{Tag: "a",
						Attrs:    []doc.ResolvedCustomAttr{{Name: "href", Value: "ftp://files.example.com/pub"}},
						Children: []doc.ResolvedCustomNode{{IsText: true, Text: "Download files"}}},
				}},
				{Tag: "p", Children: []doc.ResolvedCustomNode{
					{Tag: "a",
						Attrs:    []doc.ResolvedCustomAttr{{Name: "href", Value: "https://example.com/claim"}},
						Children: []doc.ResolvedCustomNode{{IsText: true, Text: "Visit site"}}},
				}},
			},
		}},
	}}
	want := " // \n\nDownload files\n\nVisit site (https://example.com/claim)"
	if got := Write(resolved); got != want {
		t.Errorf("Custom-link text mismatch.\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

// TestEM111TextPartAltFallbackNeverLeaksURLs is the text twin of the EM111
// omission and alt-fallback proofs: a Hero, Column, or Custom <img> whose
// src fails doc.SafeImgSrc still renders exactly its "[alt]" stand-in —
// the same fallback shape the HTML writer emits — and none of the three
// unsafe src values may appear anywhere in the text part. The text writer
// never emitted image URLs, so this pins the invariant rather than a new
// behavior: the URL-safety change must not give the text part anything
// new to leak.
func TestEM111TextPartAltFallbackNeverLeaksURLs(t *testing.T) {
	resolved := &doc.Resolved{Blocks: []doc.ResolvedBlock{
		doc.ResolvedHero{Src: unsafeHeroSrc, Alt: heroAltFallback},
		doc.ResolvedColumns{Columns: []doc.ResolvedColumn{
			{ImgSrc: unsafeColumnSrc, ImgAlt: columnAltFallback, Title: "Column title"},
		}},
		doc.ResolvedCustom{Root: doc.ResolvedCustomNode{
			Tag: "div",
			Children: []doc.ResolvedCustomNode{
				{Tag: "img",
					Attrs: []doc.ResolvedCustomAttr{{Name: "src", Value: unsafeCustomSrc}, {Name: "alt", Value: customAltFallback}}},
			},
		}},
	}}
	want := " // \n\n[" + heroAltFallback + "]\n\n[" + columnAltFallback + "]\nColumn title\n\n[" + customAltFallback + "]"
	if got := Write(resolved); got != want {
		t.Errorf("alt-fallback text mismatch.\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}

	text := Write(resolved)
	for _, unsafe := range []string{unsafeHeroSrc, unsafeColumnSrc, unsafeCustomSrc} {
		if strings.Contains(text, unsafe) {
			t.Errorf("text part leaks the rejected img src %q; an unsafe URL must reach neither output part", unsafe)
		}
	}
}
