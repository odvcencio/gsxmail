package gsxmail_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
)

// RecapProps mirrors testdata/emails/recap.go's declared props type (the
// design spec's section 6.5 worked example, the N7 "YOUR HAUL" draft
// recap).
type RecapProps struct {
	League         string
	Code           string
	Tagline        string
	PickCountLabel string
	Lede           string
	HaulHeader     []string
	Haul           []HaulRow
	HasAutoPicks   bool
	AutoPickNote   string
	BoardURL       string
	FooterNote     string
}

// HaulRow mirrors testdata/emails/recap.go's HaulRow.
type HaulRow struct {
	Cells      []string
	IsKeystone bool
}

// recapFixtureProps is the 15-row fixture testdata/recap.{html,txt}
// render from: a 15-pick Haul, one marked ("keystone") row (index 6, the
// 4.03 pick), and HasAutoPicks true so the <If>-wrapped Note also renders.
func recapFixtureProps() RecapProps {
	positions := []string{"QB", "RB", "WR", "WR", "TE", "RB", "WR", "OT", "CB", "WR", "RB", "QB", "TE", "S", "WR"}
	names := []string{
		"Josh Rivera", "Malik Fontaine", "Devon Okafor", "Trey Whitfield", "Cole Bannister",
		"Jamal Osei", "Ezra Lindqvist", "Marcus DeLuca", "Isaiah Marchetti", "Kenji Yamada",
		"Andre Castellan", "Owen Prescott", "Diego Marchetti", "Xavier Odom", "Leo Bertrand",
	}
	haul := make([]HaulRow, 15)
	for i := range haul {
		round := i/2 + 1
		pick := i%2*6 + 3
		haul[i] = HaulRow{
			Cells: []string{
				fmt.Sprintf("%d.%02d", round, pick),
				names[i],
				positions[i],
				"Aqua",
			},
			IsKeystone: i == 6,
		}
	}
	return RecapProps{
		League:         "GRIDIRON 2000",
		Code:           "G2K",
		Tagline:        "DYNASTY FANTASY LEAGUE",
		PickCountLabel: "15 PICKS",
		Lede:           "Fifteen names, one roster. The board is final and the group chat has receipts.",
		HaulHeader:     []string{"PICK", "PLAYER", "POS", "TEAM"},
		Haul:           haul,
		HasAutoPicks:   true,
		AutoPickNote:   "Two picks were on autopick after 11 PM — Ezra Lindqvist is one of them.",
		BoardURL:       "https://gridiron.draco.quest/board",
		FooterNote:     "GRIDIRON 2000 · Eight seats. One trophy. Permanent group-chat evidence.",
	}
}

func loadRecapSet(t *testing.T) *gsxmail.Set {
	t.Helper()
	set, err := gsxmail.Load(os.DirFS("testdata/emails"), gsxmail.Options{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return set
}

// TestRecapGolden is the WP3 acceptance golden: a 15-row StatTable built
// from <Each>, a mark carried through from a loop-bound struct field
// (row.IsKeystone), and an <If>-wrapped Note, all rendered from one
// EmailDoc into both parts (design spec section 6.5, section 15 WP3).
func TestRecapGolden(t *testing.T) {
	set := loadRecapSet(t)
	parts, err := set.Render("DraftRecap", recapFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	wantHTML, err := os.ReadFile("testdata/recap.html")
	if err != nil {
		t.Fatal(err)
	}
	wantText, err := os.ReadFile("testdata/recap.txt")
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

// TestRecapEmptyHaulRendersNothing proves the design constraint "empty
// slice renders nothing": an <Each> over a zero-length Haul contributes no
// StatTable rows, and the table still renders its header with none.
func TestRecapEmptyHaulRendersNothing(t *testing.T) {
	set := loadRecapSet(t)
	props := recapFixtureProps()
	props.Haul = nil
	props.HasAutoPicks = false

	parts, err := set.Render("DraftRecap", props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(parts.Text, "PICK  PLAYER") {
		t.Error("text part lost the StatTable header when Haul is empty")
	}
	if strings.Contains(parts.Text, "Two picks were on autopick") {
		t.Error("text part rendered the <If>-wrapped Note despite HasAutoPicks being false")
	}
	if strings.Contains(parts.HTML, "Two picks were on autopick") {
		t.Error("HTML part rendered the <If>-wrapped Note despite HasAutoPicks being false")
	}
}

// TestRecapDeterminism renders the same Set and props twice and requires
// byte-identical output — the same T2 guarantee WP1's invite golden pins,
// now exercised through <Each>/<If> dynamic-data resolution too.
func TestRecapDeterminism(t *testing.T) {
	set := loadRecapSet(t)
	props := recapFixtureProps()

	first, err := set.Render("DraftRecap", props)
	if err != nil {
		t.Fatalf("Render #1: %v", err)
	}
	second, err := set.Render("DraftRecap", props)
	if err != nil {
		t.Fatalf("Render #2: %v", err)
	}
	if first.HTML != second.HTML {
		t.Error("HTML differs between two renders of the same Set and props")
	}
	if first.Text != second.Text {
		t.Error("text differs between two renders of the same Set and props")
	}
}
