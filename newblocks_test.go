package gsxmail_test

import (
	"os"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
	"m31labs.dev/gsxmail/internal/structverify"
)

// NewBlocksProps mirrors testdata/newblocks/newblocks.go's declared props
// type (design spec section 15, WP5.3).
type NewBlocksProps struct {
	Wordmark  string
	ShortCode string
	Tagline   string
	Title     string

	BadgeNeutralText  string
	BadgePositiveText string
	BadgeWarningText  string
	BadgeCriticalText string

	HeroSrc    string
	HeroAlt    string
	HeroWidth  string
	HeroHeight string

	Col1ImgSrc    string
	Col1ImgAlt    string
	Col1ImgWidth  string
	Col1ImgHeight string
	Col1Title     string
	Col1Text      string
	Col2Title     string
	Col2Text      string

	SpacerHeight string

	PrimaryLabel   string
	PrimaryHref    string
	SecondaryLabel string
	SecondaryHref  string
	LinkLabel      string
	LinkHref       string
	LinkWidth      string
}

func newBlocksFixtureProps() NewBlocksProps {
	return NewBlocksProps{
		Wordmark:  "SENTINEL-WORDMARK-NB",
		ShortCode: "NBK",
		Tagline:   "SENTINEL-TAGLINE-NB",
		Title:     "SENTINEL-TITLE-NB",

		BadgeNeutralText:  "sentinel-badge-neutral",
		BadgePositiveText: "sentinel-badge-positive",
		BadgeWarningText:  "sentinel-badge-warning",
		BadgeCriticalText: "sentinel-badge-critical",

		HeroSrc:    "https://example.com/sentinel-hero@2x.png",
		HeroAlt:    "sentinel-hero-alt",
		HeroWidth:  "598",
		HeroHeight: "240",

		Col1ImgSrc:    "https://example.com/sentinel-col1@2x.png",
		Col1ImgAlt:    "sentinel-col1-img-alt",
		Col1ImgWidth:  "120",
		Col1ImgHeight: "80",
		Col1Title:     "sentinel-col1-title",
		Col1Text:      "sentinel-col1-text",
		Col2Title:     "sentinel-col2-title",
		Col2Text:      "sentinel-col2-text",

		SpacerHeight: "24",

		PrimaryLabel:   "sentinel-primary-label",
		PrimaryHref:    "https://example.com/sentinel-primary",
		SecondaryLabel: "sentinel-secondary-label",
		SecondaryHref:  "https://example.com/sentinel-secondary",
		LinkLabel:      "sentinel-link-label",
		LinkHref:       "https://example.com/sentinel-link",
		LinkWidth:      "220",
	}
}

func loadNewBlocksSet(t *testing.T, opts gsxmail.Options) *gsxmail.Set {
	t.Helper()
	set, err := gsxmail.Load(os.DirFS("testdata/newblocks"), opts)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return set
}

// TestNewBlocksSentinelsAppearInBothParts is T8-style (design spec section
// 11) for WP5.3's new components: every field renders somewhere in the
// HTML part; every field with a defined text-twin derivation (pixel
// dossier sections 4.4/4.9/4.10/4.11) renders in the text part too, and
// the HTML-only ones (widths, heights, hrefs the text twin folds into its
// own "-> LABEL: URL"/"[alt]" shapes rather than repeating verbatim) are
// asserted not to leak where they should not.
func TestNewBlocksSentinelsAppearInBothParts(t *testing.T) {
	set := loadNewBlocksSet(t, gsxmail.Options{})
	parts, err := set.Render("NewBlocks", newBlocksFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	htmlOnly := []string{
		"sentinel-badge-neutral", // wrapped as [sentinel-badge-neutral] in text
		"sentinel-hero-alt",      // wrapped as [sentinel-hero-alt] in text
		"sentinel-col1-img-alt",  // wrapped as [sentinel-col1-img-alt] in text
	}
	for _, s := range htmlOnly {
		if !strings.Contains(parts.HTML, s) {
			t.Errorf("HTML part is missing sentinel %q", s)
		}
	}

	bothParts := []string{
		"sentinel-col1-title", "sentinel-col1-text",
		"sentinel-col2-title", "sentinel-col2-text",
		"sentinel-primary-label", "sentinel-secondary-label", "sentinel-link-label",
	}
	for _, s := range bothParts {
		if !strings.Contains(parts.HTML, s) {
			t.Errorf("HTML part is missing sentinel %q", s)
		}
		if !strings.Contains(parts.Text, s) {
			t.Errorf("text part is missing sentinel %q", s)
		}
	}

	for _, url := range []string{
		"https://example.com/sentinel-primary",
		"https://example.com/sentinel-secondary",
		"https://example.com/sentinel-link",
	} {
		if !strings.Contains(parts.HTML, url) {
			t.Errorf("HTML part is missing href %q", url)
		}
		if !strings.Contains(parts.Text, url) {
			t.Errorf("text part is missing url %q", url)
		}
	}

	for _, want := range []string{
		"[sentinel-badge-neutral]", "[sentinel-badge-positive]",
		"[sentinel-badge-warning]", "[sentinel-badge-critical]",
	} {
		if !strings.Contains(parts.Text, want) {
			t.Errorf("text part is missing bracketed Badge %q", want)
		}
	}
	if !strings.Contains(parts.Text, "[sentinel-hero-alt]") {
		t.Error("text part is missing the bracketed Hero alt")
	}
	if !strings.Contains(parts.Text, "[sentinel-col1-img-alt]") {
		t.Error("text part is missing the bracketed Column image alt")
	}

	if !strings.Contains(parts.HTML, `width="598"`) || !strings.Contains(parts.HTML, `height="240"`) {
		t.Error("HTML part is missing Hero's width/height attributes")
	}
	if strings.Contains(parts.Text, "598") || strings.Contains(parts.Text, "240") {
		t.Error("text part leaked Hero's width/height pixel values")
	}
}

// TestNewBlocksStructuralPass proves WP5.3's new components clear the
// existing structural verification pass (EM131-EM134; hardened mode also
// gets VerifyContract's role="presentation" table-split proof), in both
// output contracts, without needing any new structverify code — the
// pass's existing well-formedness and conditional-comment-balance checks
// already generalize to Button's "link" variant's [if mso] glyph-spacing
// hack and Columns' own ghost table.
func TestNewBlocksStructuralPass(t *testing.T) {
	props := newBlocksFixtureProps()

	hardSet := loadNewBlocksSet(t, gsxmail.Options{})
	hardParts, err := hardSet.Render("NewBlocks", props)
	if err != nil {
		t.Fatalf("Render (hardened): %v", err)
	}
	findings, err := structverify.VerifyContract(hardParts.HTML)
	if err != nil {
		t.Fatalf("VerifyContract (hardened): %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("NewBlocks (hardened) fails the structural pass:\n%s", joinFindings(findings))
	}

	paritySet := loadNewBlocksSet(t, gsxmail.Options{Outlook: "off"})
	parityParts, err := paritySet.Render("NewBlocks", props)
	if err != nil {
		t.Fatalf("Render (parity): %v", err)
	}
	findings, err = structverify.Verify(parityParts.HTML)
	if err != nil {
		t.Fatalf("Verify (parity): %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("NewBlocks (parity) is not well-formed:\n%s", joinFindings(findings))
	}
}

// TestNewBlocksGolden pins NewBlocks' hardened-mode byte output (design
// spec section 15, WP5.3): the same T1-style golden-pair guarantee every
// other stdlib block corpus carries.
func TestNewBlocksGolden(t *testing.T) {
	set := loadNewBlocksSet(t, gsxmail.Options{})
	parts, err := set.Render("NewBlocks", newBlocksFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	wantHTML, err := os.ReadFile("testdata/newblocks.html")
	if err != nil {
		t.Fatal(err)
	}
	wantText, err := os.ReadFile("testdata/newblocks.txt")
	if err != nil {
		t.Fatal(err)
	}
	if parts.HTML != string(wantHTML) {
		t.Errorf("HTML part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.HTML, wantHTML)
	}
	if parts.Text != string(wantText) {
		t.Errorf("text part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.Text, wantText)
	}
}

// TestNewBlocksParityGolden pins NewBlocks' parity-mode ("off") byte
// output: the new components carry no WP1 legacy, so parity mode differs
// from hardened only in the Shell frame (no ghost table, no DPI
// namespaces) — the components themselves render identically in both
// contracts (renderhtml's own writeHero/writeSpacer/writeBadge/
// writeButtonLink/writeButtonSecondary doc comments state this).
func TestNewBlocksParityGolden(t *testing.T) {
	set := loadNewBlocksSet(t, gsxmail.Options{Outlook: "off"})
	parts, err := set.Render("NewBlocks", newBlocksFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	wantHTML, err := os.ReadFile("testdata/newblocks_parity.html")
	if err != nil {
		t.Fatal(err)
	}
	if parts.HTML != string(wantHTML) {
		t.Errorf("HTML part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.HTML, wantHTML)
	}
}
