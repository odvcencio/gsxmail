package main

import (
	"testing"

	"m31labs.dev/gsxmail"
)

// TestSortDiagnostics is polish item 9's own proof: findings sort by
// file, then line, then column, so every finding in one file groups
// together, in source order.
func TestSortDiagnostics(t *testing.T) {
	in := []gsxmail.Diagnostic{
		{File: "b.gsx", Line: 1, Col: 1, Code: "EM001"},
		{File: "a.gsx", Line: 5, Col: 1, Code: "EM002"},
		{File: "a.gsx", Line: 2, Col: 9, Code: "EM003"},
		{File: "a.gsx", Line: 2, Col: 3, Code: "EM004"},
	}
	got := sortDiagnostics(in)
	want := []string{"EM004", "EM003", "EM002", "EM001"}
	for i, code := range want {
		if got[i].Code != code {
			t.Errorf("got[%d].Code = %q, want %q (full order: %+v)", i, got[i].Code, code, got)
		}
	}
}

// TestFilterSeverity is --severity's own proof.
func TestFilterSeverity(t *testing.T) {
	diags := []gsxmail.Diagnostic{
		{Code: "EM101", Severity: "warn"},
		{Code: "EM001", Severity: "error"},
	}
	if got := filterSeverity(diags, "all"); len(got) != 2 {
		t.Errorf("filterSeverity(all) = %d diags, want 2", len(got))
	}
	if got := filterSeverity(diags, "warn"); len(got) != 2 {
		t.Errorf("filterSeverity(warn) = %d diags, want 2 (warn keeps warn and error)", len(got))
	}
	got := filterSeverity(diags, "error")
	if len(got) != 1 || got[0].Code != "EM001" {
		t.Errorf("filterSeverity(error) = %+v, want just EM001", got)
	}
}
