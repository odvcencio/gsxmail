package gsxmail_test

import (
	"os"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
)

// AllBlocksProps mirrors testdata/allblocks/allblocks.go's declared props
// type.
type AllBlocksProps struct {
	Wordmark   string
	ShortCode  string
	Tagline    string
	Title      string
	Preheader  string
	ShoutName  string
	HeadTitle  string
	HeadLede   string
	PanelLabel string
	PanelValue string
	CTALabel   string
	CTAHref    string
	PickTitle  string
	PickItem   string
	StatTitle  string
	StatHeader []string
	StatRows   []AllBlocksRow
	ShowNote   bool
	NoteText   string
	LinkLabel  string
	LinkHref   string
	ImgAlt     string
	ImgSrc     string
	TableCell  string
	FooterSign string
	FooterNote string
}

// AllBlocksRow mirrors testdata/allblocks/allblocks.go's AllBlocksRow.
type AllBlocksRow struct {
	Cells []string
	Mark  bool
}

// shoutHelper is AllBlocks' one registered Options.Helpers entry: a pure
// func over scalar types, per Options.Helpers' own doc comment.
func shoutHelper(name string) string {
	return strings.ToUpper(name) + "!!!"
}

func allBlocksFixtureProps() AllBlocksProps {
	return AllBlocksProps{
		Wordmark:   "SENTINEL-WORDMARK",
		ShortCode:  "SNT",
		Tagline:    "SENTINEL-TAGLINE",
		Title:      "SENTINEL-TITLE",
		Preheader:  "sentinel-preheader-text",
		ShoutName:  "sentinel-shout",
		HeadTitle:  "SENTINEL-HEAD-TITLE",
		HeadLede:   "sentinel-head-lede",
		PanelLabel: "SENTINEL-PANEL-LABEL",
		PanelValue: "sentinel-panel-value",
		CTALabel:   "SENTINEL-CTA-LABEL",
		CTAHref:    "https://example.com/sentinel-cta",
		PickTitle:  "SENTINEL-PICK-TITLE",
		PickItem:   "sentinel-pick-item",
		StatTitle:  "SENTINEL-STAT-TITLE",
		StatHeader: []string{"SENTINEL-COL-A", "SENTINEL-COL-B"},
		StatRows: []AllBlocksRow{
			{Cells: []string{"sentinel-row1-a", "sentinel-row1-b"}, Mark: false},
			{Cells: []string{"sentinel-row2-a", "sentinel-row2-b"}, Mark: true},
		},
		ShowNote:   true,
		NoteText:   "sentinel-note-text",
		LinkLabel:  "sentinel-link-label",
		LinkHref:   "https://example.com/sentinel-link",
		ImgAlt:     "sentinel-img-alt",
		ImgSrc:     "https://example.com/sentinel.png",
		TableCell:  "sentinel-table-cell",
		FooterSign: "sentinel-footer-sign",
		FooterNote: "sentinel-footer-note",
	}
}

// TestAllBlocksSentinelsAppearInBothParts is T8 (design spec section 11):
// every stdlib block, plus the Custom raw-element escape hatch and a
// registered helper call, renders every sentinel prop value into both the
// HTML and text parts (emailkit rule 4, inherited structurally).
func TestAllBlocksSentinelsAppearInBothParts(t *testing.T) {
	set, err := gsxmail.Load(os.DirFS("testdata/allblocks"), gsxmail.Options{
		Helpers: map[string]any{"shout": shoutHelper},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	parts, err := set.Render("AllBlocks", allBlocksFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// ShortCode ("SNT") is HTML-only by design (the header badge; WP1's
	// rendertext.Write never included it, matching emailkit's own text
	// renderer), so it is checked separately below rather than through
	// this both-parts loop.
	if !strings.Contains(parts.HTML, "SNT") {
		t.Error("HTML part is missing the shortCode sentinel")
	}

	// Preheader (design spec section 15, WP5.2) is HTML-only, like
	// ShortCode: the hidden inbox-preview div has no plain-text
	// equivalent, and rendertext never reads doc.ResolvedShell.Preheader.
	if !strings.Contains(parts.HTML, "sentinel-preheader-text") {
		t.Error("HTML part is missing the preheader sentinel")
	}
	if strings.Contains(parts.Text, "sentinel-preheader-text") {
		t.Error("text part rendered the preheader; it is an HTML-only field")
	}

	sentinels := []string{
		"SENTINEL-WORDMARK",
		"SENTINEL-TAGLINE",
		"SENTINEL-HEAD-TITLE",
		"sentinel-head-lede",
		"SENTINEL-PANEL-LABEL",
		"sentinel-panel-value",
		"SENTINEL-CTA-LABEL",
		"SENTINEL-PICK-TITLE",
		"sentinel-pick-item",
		"SENTINEL-STAT-TITLE",
		"SENTINEL-COL-A",
		"SENTINEL-COL-B",
		"sentinel-row1-a",
		"sentinel-row1-b",
		"sentinel-row2-a",
		"sentinel-row2-b",
		"sentinel-note-text",
		"sentinel-link-label",
		"sentinel-table-cell",
		"sentinel-footer-sign",
		"sentinel-footer-note",
		shoutHelper("sentinel-shout"), // proves the helper actually ran
	}
	for _, s := range sentinels {
		if !strings.Contains(parts.HTML, s) {
			t.Errorf("HTML part is missing sentinel %q", s)
		}
		if !strings.Contains(parts.Text, s) {
			t.Errorf("text part is missing sentinel %q", s)
		}
	}

	// The href-only sentinels: the HTML part carries them as an href
	// attribute value; the text twin carries the CTA's as "-> LABEL: URL"
	// and the Custom <a>'s as "label (url)" (design spec section 9).
	for _, url := range []string{"https://example.com/sentinel-cta", "https://example.com/sentinel-link"} {
		if !strings.Contains(parts.HTML, url) {
			t.Errorf("HTML part is missing href %q", url)
		}
		if !strings.Contains(parts.Text, url) {
			t.Errorf("text part is missing url %q", url)
		}
	}

	// The <img>'s alt text renders as visible text in the HTML img
	// attribute and as "[alt]" in the Custom subtree's derived text.
	if !strings.Contains(parts.HTML, `alt="sentinel-img-alt"`) {
		t.Error("HTML part is missing the img alt attribute")
	}
	if !strings.Contains(parts.Text, "[sentinel-img-alt]") {
		t.Error("text part is missing the derived [alt] form of the Custom <img>")
	}

	// The marked StatTable row (index 1, sentinel-row2-*) carries a "* "
	// prefix in text and the accent color in HTML (MarkRow, 1-based).
	if !strings.Contains(parts.Text, "* sentinel-row2-a") {
		t.Error("text part did not mark the second StatTable row")
	}
}

// TestAllBlocksSkipsDivider proves Divider produces no visible sentinel of
// its own (it has none) without disturbing its neighbors' blank-line
// spacing: exactly one blank line separates the blocks on either side of
// it, not three (the emailkit nit-1 finding, ported).
func TestAllBlocksSkipsDivider(t *testing.T) {
	set, err := gsxmail.Load(os.DirFS("testdata/allblocks"), gsxmail.Options{
		Helpers: map[string]any{"shout": shoutHelper},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	parts, err := set.Render("AllBlocks", allBlocksFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(parts.Text, "\n\n\n") {
		t.Error("text part has a triple-blank-line gap; Divider's own separator was not skipped")
	}
}
