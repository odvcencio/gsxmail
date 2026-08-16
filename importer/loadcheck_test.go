package importer

import (
	"encoding/json"
	"testing"
	"testing/fstest"

	"m31labs.dev/gsxmail"
)

// loadAndRender writes res's generated .gsx/props.go into an in-memory
// fs.FS, loads it through gsxmail.Load, and renders the one declared
// template against props.sample.json's own decoded map — proof-bar item
// (c): "every imported .gsx must LOAD cleanly through gsxmail.Load
// (lint-clean or with only warn-severity findings) and render without
// error using the sample props." It is a shared helper: every test file
// in this package that needs a real Load+Render round trip on a Result
// calls it, rather than reimplementing the fstest.MapFS wiring.
func loadAndRender(t *testing.T, res *Result, templateName string) (gsxmail.Parts, []gsxmail.Diagnostic) {
	t.Helper()
	fsys := fstest.MapFS{
		"template.gsx": {Data: []byte(res.TemplateGSX)},
		"props.go":     {Data: []byte(res.PropsGo)},
	}
	set, err := gsxmail.Load(fsys, gsxmail.Options{})
	if err != nil {
		t.Fatalf("gsxmail.Load: %v\n--- template.gsx ---\n%s\n--- props.go ---\n%s", err, res.TemplateGSX, res.PropsGo)
	}
	diags := set.Check()
	for _, d := range diags {
		if d.Severity == "error" {
			t.Fatalf("Load produced an error-severity finding: %s", d.String())
		}
	}

	var props map[string]any
	if err := json.Unmarshal([]byte(res.SamplePropsJSON), &props); err != nil {
		t.Fatalf("decoding props.sample.json: %v", err)
	}
	parts, err := set.Render(templateName, props)
	if err != nil {
		t.Fatalf("Render: %v\n--- template.gsx ---\n%s", err, res.TemplateGSX)
	}
	return parts, diags
}

// loadAndRenderHTML is loadAndRender's own thin wrapper returning just
// the rendered HTML string — the idempotence test's own
// "render(import(x))" step.
func loadAndRenderHTML(t *testing.T, res *Result, templateName string) (gsxmail.Parts, string) {
	t.Helper()
	parts, _ := loadAndRender(t, res, templateName)
	return parts, parts.HTML
}
