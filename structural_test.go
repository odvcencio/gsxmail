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
