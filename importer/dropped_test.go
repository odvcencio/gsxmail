package importer

import (
	"strings"
	"testing"
)

// TestDroppedTagsAreReported is the launch-gate m16 finding's own proof:
// a <script> in the source (something dropEntirely, not preserved
// anywhere in the generated template) now shows up in Report.Dropped and
// in IMPORT-REPORT.md's own "Dropped entirely" section — the one case
// "a node it cannot confidently place never gets dropped" does not
// cover.
func TestDroppedTagsAreReported(t *testing.T) {
	src := `<html><body>
<table width="600" style="max-width:600px;"><tr><td>
<div>Some card content that is not a recognized component shape at all, just plain filler prose text here.</div>
<script>alert('tracking pixel loader');</script>
</td></tr></table>
</body></html>`

	res, err := Import([]byte(src), "dropped.html", Options{TemplateName: "Dropped"})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if res.Report.Dropped["script"] == 0 {
		t.Fatalf("Report.Dropped[\"script\"] = %d, want at least 1; got Dropped = %+v", res.Report.Dropped["script"], res.Report.Dropped)
	}

	md := res.Report.WriteMarkdown()
	if !strings.Contains(md, "## Dropped entirely") {
		t.Error("IMPORT-REPORT.md is missing the \"Dropped entirely\" section")
	}
	if !strings.Contains(md, "`<script>`") {
		t.Error("IMPORT-REPORT.md's \"Dropped entirely\" section does not name <script>")
	}
	if strings.Contains(res.TemplateGSX, "alert(") {
		t.Error("template.gsx still carries the script's own content; it should have been dropped, not preserved")
	}
}

// TestNoDroppedTagsOmitsSection is the negative case: a source with
// nothing in dropEntirely gets no "Dropped entirely" section at all —
// the report stays exactly as it read before m16, for every fixture
// that never trips this case.
func TestNoDroppedTagsOmitsSection(t *testing.T) {
	src := `<html><body>
<table width="600" style="max-width:600px;"><tr><td>
<div>Some card content that is not a recognized component shape at all, just plain filler prose text here.</div>
</td></tr></table>
</body></html>`

	res, err := Import([]byte(src), "clean.html", Options{TemplateName: "Clean"})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(res.Report.Dropped) != 0 {
		t.Errorf("Report.Dropped = %+v, want none", res.Report.Dropped)
	}
	if strings.Contains(res.Report.WriteMarkdown(), "## Dropped entirely") {
		t.Error("IMPORT-REPORT.md has a \"Dropped entirely\" section with nothing dropped")
	}
}
