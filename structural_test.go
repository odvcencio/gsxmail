package gsxmail_test

import (
	"os"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
	"m31labs.dev/gsxmail/internal/structverify"
)

// joinFindings renders a structverify.Finding slice as one message-per-
// line string for a test failure.
func joinFindings(findings []structverify.Finding) string {
	lines := make([]string, len(findings))
	for i, f := range findings {
		lines[i] = f.String()
	}
	return strings.Join(lines, "\n")
}

// TestStructuralPassOnGoldens re-parses every checked-in golden HTML file
// with gotreesitter's HTML grammar (design spec section 15, WP5.1's
// structural verification pass; pixel dossier section 7.2) and proves it
// re-parses cleanly. testdata/invite.html renders in parity mode (its own
// T3 DOM-parity guard, parity_test.go, pins it there) and gets the
// well-formedness-only check; testdata/recap.html renders hardened and
// gets the full WP5.1 output-contract check too.
func TestStructuralPassOnGoldens(t *testing.T) {
	invite, err := os.ReadFile("testdata/invite.html")
	if err != nil {
		t.Fatal(err)
	}
	findings, err := structverify.Verify(string(invite))
	if err != nil {
		t.Fatalf("Verify(invite.html): %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("invite.html (parity mode) is not well-formed:\n%s", joinFindings(findings))
	}

	recap, err := os.ReadFile("testdata/recap.html")
	if err != nil {
		t.Fatal(err)
	}
	findings, err = structverify.VerifyContract(string(recap))
	if err != nil {
		t.Fatalf("VerifyContract(recap.html): %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("recap.html (hardened mode) fails the WP5.1 output contract:\n%s", joinFindings(findings))
	}
}

// TestStructuralPassOnBlockCorpus renders testdata/allblocks — every
// stdlib block, plus the Custom raw-element escape hatch and a
// registered helper call (design spec section 11, T8) — in both output
// contracts and re-parses each with the structural verification pass:
// hardened mode must clear the full WP5.1 contract check; parity mode
// must still be well-formed HTML.
func TestStructuralPassOnBlockCorpus(t *testing.T) {
	props := allBlocksFixtureProps()
	helpers := map[string]any{"shout": shoutHelper}

	hardSet, err := gsxmail.Load(os.DirFS("testdata/allblocks"), gsxmail.Options{Helpers: helpers})
	if err != nil {
		t.Fatalf("Load (hardened): %v", err)
	}
	hardParts, err := hardSet.Render("AllBlocks", props)
	if err != nil {
		t.Fatalf("Render (hardened): %v", err)
	}
	findings, err := structverify.VerifyContract(hardParts.HTML)
	if err != nil {
		t.Fatalf("VerifyContract(AllBlocks hardened): %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("AllBlocks (hardened mode) fails the WP5.1 output contract:\n%s", joinFindings(findings))
	}

	paritySet, err := gsxmail.Load(os.DirFS("testdata/allblocks"), gsxmail.Options{Helpers: helpers, Outlook: "off"})
	if err != nil {
		t.Fatalf("Load (parity): %v", err)
	}
	parityParts, err := paritySet.Render("AllBlocks", props)
	if err != nil {
		t.Fatalf("Render (parity): %v", err)
	}
	findings, err = structverify.Verify(parityParts.HTML)
	if err != nil {
		t.Fatalf("Verify(AllBlocks parity): %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("AllBlocks (parity mode) is not well-formed:\n%s", joinFindings(findings))
	}
}

// TestStructuralPreheaderStackCatchesMissingStyles is EM173's negative
// case (design spec section 15, WP5.2): a first-child-of-body div that
// looks like a preheader (display:none first) but is missing one of the
// other five required suppression declarations must fail the pass, not
// silently pass because it merely resembles react-email's shape.
func TestStructuralPreheaderStackCatchesMissingStyles(t *testing.T) {
	html := `<!DOCTYPE html>
<html><head><title>x</title></head>
<body>
<div style="display:none; overflow:hidden; line-height:1px;">preview text</div>
<table><tr><td>hi</td></tr></table>
</body>
</html>`
	findings, err := structverify.Verify(html)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var found bool
	for _, f := range findings {
		if f.Code == "EM173" {
			found = true
		}
	}
	if !found {
		t.Errorf("EM173 did not fire on a preheader div missing opacity/max-height/max-width:\n%s", joinFindings(findings))
	}
}

// TestStructuralPreheaderStackIgnoresOrdinaryHiddenDiv proves EM173 only
// judges a div that already looks like a preheader (display:none as its
// first declared property): an ordinary hidden div elsewhere is not this
// check's business, and a template with no preheader configured at all
// (renderhtml.writePreheader's no-op case) must never trip it.
func TestStructuralPreheaderStackIgnoresOrdinaryHiddenDiv(t *testing.T) {
	html := `<!DOCTYPE html>
<html><head><title>x</title></head>
<body>
<table><tr><td style="display:none;">not a preheader</td></tr></table>
</body>
</html>`
	findings, err := structverify.Verify(html)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, f := range findings {
		if f.Code == "EM173" {
			t.Errorf("EM173 fired on a template with no preheader-shaped div: %s", f)
		}
	}
}

// TestStructuralDarkStyleLayerCatchesMissingHooks is EM174's negative case
// (design spec section 15, WP5.2): a <style> block that declares
// prefers-color-scheme but drops one of the [data-ogsc]/[data-ogsb]
// Outlook-app hooks, or unbalances its braces, must fail the pass.
func TestStructuralDarkStyleLayerCatchesMissingHooks(t *testing.T) {
	html := `<!DOCTYPE html>
<html><head><title>x</title>
<style>
@media (prefers-color-scheme: dark) {
.gsx-card { background-color:#101014 !important; }
}
</style>
</head>
<body><table><tr><td>hi</td></tr></table></body>
</html>`
	findings, err := structverify.Verify(html)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var found bool
	for _, f := range findings {
		if f.Code == "EM174" {
			found = true
		}
	}
	if !found {
		t.Errorf("EM174 did not fire on a dark style layer missing both [data-ogsc] and [data-ogsb]:\n%s", joinFindings(findings))
	}
}

// TestStructuralDarkStyleLayerIgnoresLightStyle proves EM174 only judges
// a <style> block that actually declares prefers-color-scheme: a "none"
// or "locked" template's plain reset/media-query style block must never
// trip it.
func TestStructuralDarkStyleLayerIgnoresLightStyle(t *testing.T) {
	html := `<!DOCTYPE html>
<html><head><title>x</title>
<style>
body{margin:0;}
@media only screen and (max-width:480px){.gsx-card{width:100% !important;}}
</style>
</head>
<body><table><tr><td>hi</td></tr></table></body>
</html>`
	findings, err := structverify.Verify(html)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, f := range findings {
		if f.Code == "EM174" {
			t.Errorf("EM174 fired on a style block with no prefers-color-scheme declaration: %s", f)
		}
	}
}

// TestStructuralPassOnInviteAndRecapRenders re-parses InviteEmail (parity
// mode, per its own DOM-parity guard) and DraftRecap (hardened) freshly
// rendered through Set.Render, not just their golden files, so the pass
// runs on the exact code path any consumer's Render call uses.
func TestStructuralPassOnInviteAndRecapRenders(t *testing.T) {
	inviteSet := loadInviteSet(t)
	inviteParts, err := inviteSet.Render("InviteEmail", inviteFixtureProps())
	if err != nil {
		t.Fatalf("Render InviteEmail: %v", err)
	}
	findings, err := structverify.Verify(inviteParts.HTML)
	if err != nil {
		t.Fatalf("Verify(InviteEmail): %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("InviteEmail (parity mode) is not well-formed:\n%s", joinFindings(findings))
	}

	recapSet := loadRecapSet(t)
	recapParts, err := recapSet.Render("DraftRecap", recapFixtureProps())
	if err != nil {
		t.Fatalf("Render DraftRecap: %v", err)
	}
	findings, err = structverify.VerifyContract(recapParts.HTML)
	if err != nil {
		t.Fatalf("VerifyContract(DraftRecap): %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("DraftRecap (hardened mode) fails the WP5.1 output contract:\n%s", joinFindings(findings))
	}
}
