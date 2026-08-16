package gsxmail_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"m31labs.dev/gsxmail"
)

// BigStatsProps mirrors testdata/emails/bigstats.go's declared props type:
// an arbitrary-length StatTable whose row count the test controls, so one
// template can cross both the EM121 warning line and the EM120 budget
// (design spec section 11, T6).
type BigStatsProps struct {
	Wordmark string
	Header   []string
	Rows     []BigRow
}

// BigRow mirrors testdata/emails/bigstats.go's BigRow.
type BigRow struct {
	Cells []string
}

// bigStatsProps builds n StatTable rows. Rendered HTML size is linear in
// n (byte size ≈ 2585 + 853n, measured against testdata/emails/bigstats.gsx);
// the row counts TestSizeBudgetWarnsAtEM121 and TestSizeBudgetFailsAtEM120
// use carry several kilobytes of margin on both sides of the 90,000- and
// 100,000-byte lines, so a small template or writer change does not flip
// the test's outcome.
func bigStatsProps(n int) BigStatsProps {
	header := []string{"PLAYER", "POS", "TEAM", "PICK"}
	rows := make([]BigRow, n)
	for i := range rows {
		rows[i] = BigRow{Cells: []string{
			fmt.Sprintf("Player Name Number %d Extra Padding Text", i),
			"WR",
			"Some Team Name FC",
			fmt.Sprintf("%d.%02d", i/10+1, i%10+1),
		}}
	}
	return BigStatsProps{Wordmark: "BIG TEST", Header: header, Rows: rows}
}

func loadBigStatsSet(t *testing.T, opts gsxmail.Options) *gsxmail.Set {
	t.Helper()
	set, err := gsxmail.Load(os.DirFS("testdata/emails"), opts)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return set
}

// T6: a 108-row StatTable's HTML part lands between the fixed 90,000-byte
// warning line and the default 100,000-byte budget. Render succeeds and
// reports one EM121 warning in Parts.Diagnostics, with the design spec's
// exact message (section 8).
func TestSizeBudgetWarnsAtEM121(t *testing.T) {
	set := loadBigStatsSet(t, gsxmail.Options{})
	parts, err := set.Render("BigStats", bigStatsProps(108))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(parts.HTML) <= 90_000 || len(parts.HTML) > 100_000 {
		t.Fatalf("fixture does not land in the EM121 warning band: %d bytes", len(parts.HTML))
	}

	var found *gsxmail.Diagnostic
	for i := range parts.Diagnostics {
		if parts.Diagnostics[i].Code == "EM121" {
			found = &parts.Diagnostics[i]
		}
	}
	if found == nil {
		t.Fatalf("no EM121 diagnostic in Parts.Diagnostics: %+v", parts.Diagnostics)
	}
	if found.Severity != "warn" {
		t.Errorf("EM121 severity = %q, want warn", found.Severity)
	}
	want := fmt.Sprintf(
		"rendered HTML is %d bytes with fixture BigStats; within budget but above the 90000-byte warning line",
		len(parts.HTML))
	if found.Message != want {
		t.Errorf("EM121 message = %q, want %q", found.Message, want)
	}
}

// T6: a 130-row StatTable crosses the default 100,000-byte budget
// outright (design spec section 11: "a generated 130-row StatTable
// fixture crosses 100 KB and fails EM120"). Render fails closed with a
// *gsxmail.SizeBudgetError carrying EM120's exact message and a zero
// Parts.
func TestSizeBudgetFailsAtEM120(t *testing.T) {
	set := loadBigStatsSet(t, gsxmail.Options{})
	parts, err := set.Render("BigStats", bigStatsProps(130))
	if err == nil {
		t.Fatal("Render succeeded; a 130-row StatTable is supposed to cross the 100,000-byte budget")
	}
	if parts.HTML != "" || parts.Text != "" || len(parts.Diagnostics) != 0 {
		t.Errorf("Render returned a non-zero Parts alongside its error: %+v", parts)
	}

	var sizeErr *gsxmail.SizeBudgetError
	if !errors.As(err, &sizeErr) {
		t.Fatalf("Render's error is not a *gsxmail.SizeBudgetError: %v", err)
	}
	if sizeErr.Diagnostic.Code != "EM120" {
		t.Errorf("Diagnostic.Code = %q, want EM120", sizeErr.Diagnostic.Code)
	}
	if sizeErr.Diagnostic.Severity != "error" {
		t.Errorf("Diagnostic.Severity = %q, want error", sizeErr.Diagnostic.Severity)
	}
	wantPrefix := "gsxmail: EM120: rendered HTML is "
	if got := sizeErr.Error(); len(got) < len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
		t.Errorf("SizeBudgetError.Error() = %q, want a prefix of %q", got, wantPrefix)
	}
}

// MaxHTMLBytes: -1 disables both the error and the warning check.
func TestSizeBudgetDisabled(t *testing.T) {
	set := loadBigStatsSet(t, gsxmail.Options{MaxHTMLBytes: -1})
	parts, err := set.Render("BigStats", bigStatsProps(130))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(parts.Diagnostics) != 0 {
		t.Errorf("Diagnostics = %+v, want none with the budget check disabled", parts.Diagnostics)
	}
}

// A configured MaxHTMLBytes fails at that budget, not the default 100,000
// (design spec: "configurable").
func TestSizeBudgetConfigurable(t *testing.T) {
	set := loadBigStatsSet(t, gsxmail.Options{MaxHTMLBytes: 50_000})
	_, err := set.Render("BigStats", bigStatsProps(70))
	if err == nil {
		t.Fatal("Render succeeded; a 50,000-byte budget should have failed well before the default 100,000 bytes")
	}
	var sizeErr *gsxmail.SizeBudgetError
	if !errors.As(err, &sizeErr) {
		t.Fatalf("Render's error is not a *gsxmail.SizeBudgetError: %v", err)
	}
	wantSuffix := fmt.Sprintf("the budget is %d bytes (Gmail clips near 102400)", 50_000)
	if got := sizeErr.Diagnostic.Message; len(got) < len(wantSuffix) || got[len(got)-len(wantSuffix):] != wantSuffix {
		t.Errorf("Diagnostic.Message = %q, want a suffix of %q", got, wantSuffix)
	}
}
