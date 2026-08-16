// Package importer's foreign-corpus suite is the proof bar's item (b)
// (task instructions): three checked-in fixture files this package's own
// testdata/corpus holds — one MJML-compiled shape (mjml.html, citing R6-
// R9), one react-email-style shape (react-email.html, citing R3-R5), and
// one deliberately crufty legacy table-soup mail with unclosed tags
// (legacy.html) — each with a pinned .gsx and report golden, each
// required to load and render cleanly with its own harvested sample
// props (proof-bar item (c)).
package importer

import (
	"os"
	"path/filepath"
	"testing"
)

// corpusCase is one foreign-corpus fixture's own wiring: its source
// file, the template name to derive, and its golden directory.
type corpusCase struct {
	name   string // fixture base name, also this case's own subtest name
	source string // testdata/corpus/<source>
	golden string // testdata/corpus/<golden>
	tmpl   string
}

func corpusCases() []corpusCase {
	return []corpusCase{
		{name: "mjml", source: "mjml.html", golden: "mjml-golden", tmpl: "Digest"},
		{name: "react-email", source: "react-email.html", golden: "react-email-golden", tmpl: "MagicLink"},
		{name: "legacy", source: "legacy.html", golden: "legacy-golden", tmpl: "Newsletter"},
	}
}

// TestForeignCorpusGoldens pins Import's own four generated outputs
// (template.gsx, props.go, props.sample.json, IMPORT-REPORT.md) against
// this package's own checked-in golden files for each corpus fixture —
// "whatever the importer produces" is the golden, exactly like every
// other golden test in this repo (examples/gallery/gallery_test.go's own
// convention): a deliberate mapper change that improves or changes the
// output updates these files, reviewed like any other diff.
func TestForeignCorpusGoldens(t *testing.T) {
	for _, tc := range corpusCases() {
		t.Run(tc.name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("testdata", "corpus", tc.source))
			if err != nil {
				t.Fatal(err)
			}
			res, err := Import(src, tc.source, Options{TemplateName: tc.tmpl})
			if err != nil {
				t.Fatalf("Import: %v", err)
			}

			assertGolden(t, filepath.Join("testdata", "corpus", tc.golden, "template.gsx"), res.TemplateGSX)
			assertGolden(t, filepath.Join("testdata", "corpus", tc.golden, "props.go"), res.PropsGo)
			assertGolden(t, filepath.Join("testdata", "corpus", tc.golden, "props.sample.json"), res.SamplePropsJSON)
			assertGolden(t, filepath.Join("testdata", "corpus", tc.golden, "IMPORT-REPORT.md"), res.Report.WriteMarkdown())
		})
	}
}

func assertGolden(t *testing.T, path, got string) {
	t.Helper()
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v", path, err)
	}
	if got != string(want) {
		t.Errorf("%s does not match its golden.\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

// TestForeignCorpusLoadsAndRenders is proof-bar item (c) applied to the
// foreign corpus: every imported .gsx must load through gsxmail.Load
// with no error-severity finding, and render without error using its
// own harvested sample props — "the imported template is immediately
// usable, that is the product promise."
func TestForeignCorpusLoadsAndRenders(t *testing.T) {
	for _, tc := range corpusCases() {
		t.Run(tc.name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("testdata", "corpus", tc.source))
			if err != nil {
				t.Fatal(err)
			}
			res, err := Import(src, tc.source, Options{TemplateName: tc.tmpl})
			if err != nil {
				t.Fatalf("Import: %v", err)
			}
			parts, diags := loadAndRender(t, res, res.TemplateName)
			if len(parts.HTML) == 0 {
				t.Fatal("rendered HTML is empty")
			}
			for _, d := range diags {
				t.Logf("warn: %s", d.String())
			}
		})
	}
}
