// Package importer_test's self-round-trip suite is the proof bar's item
// (a) (task instructions): render the shipped gallery templates to HTML,
// import each back, and assert the importer recovers the right block
// sequence — structure fidelity, not byte fidelity.
package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"m31labs.dev/gsxmail"
	"m31labs.dev/gsxmail/examples/gallery/alert"
	"m31labs.dev/gsxmail/examples/gallery/digest"
	"m31labs.dev/gsxmail/examples/gallery/magiclink"
	"m31labs.dev/gsxmail/examples/gallery/receipt"
	"m31labs.dev/gsxmail/examples/gallery/welcome"
)

// selfRoundTripCase is one gallery template's own rendering wiring
// (mirrors examples/gallery/gallery_test.go's own galleryCase) plus the
// block-kind sequence its .gsx source declares, in order — the sequence
// Import must recover exactly.
type selfRoundTripCase struct {
	dir      string
	template string
	theme    gsxmail.Theme
	props    func() (any, error)
	want     []string
}

func loadFixture(dir, file string, out any) error {
	data, err := os.ReadFile(filepath.Join("..", "examples", "gallery", dir, file))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func selfRoundTripCases() []selfRoundTripCase {
	return []selfRoundTripCase{
		{
			dir: "welcome", template: "WelcomeEmail", theme: gsxmail.DefaultTheme(),
			props: func() (any, error) {
				var p welcome.WelcomeProps
				err := loadFixture("welcome", "welcome.props.json", &p)
				return p, err
			},
			want: []string{"Headline", "PickList", "Button", "Footer"},
		},
		{
			dir: "magiclink", template: "MagicLinkEmail", theme: gsxmail.DefaultTheme(),
			props: func() (any, error) {
				var p magiclink.MagicLinkProps
				err := loadFixture("magiclink", "magiclink.props.json", &p)
				return p, err
			},
			want: []string{"Headline", "Panel", "Note", "Button"},
		},
		{
			dir: "receipt", template: "ReceiptEmail", theme: gsxmail.DefaultTheme(),
			props: func() (any, error) {
				var p receipt.ReceiptProps
				err := loadFixture("receipt", "receipt.props.json", &p)
				return p, err
			},
			want: []string{"Badge", "Headline", "StatTable", "Panel", "Button", "Footer"},
		},
		{
			dir: "digest", template: "DigestEmail", theme: gsxmail.LedgerTheme(),
			props: func() (any, error) {
				var p digest.DigestProps
				err := loadFixture("digest", "digest.props.json", &p)
				return p, err
			},
			want: []string{"Hero", "Headline", "Columns", "Divider", "StatTable", "PickList"},
		},
		{
			dir: "alert", template: "AlertEmail", theme: gsxmail.TerminalTheme(),
			props: func() (any, error) {
				var p alert.AlertProps
				err := loadFixture("alert", "alert.props.json", &p)
				return p, err
			},
			want: []string{"Signal", "Badge", "Note", "Button"},
		},
	}
}

// renderGallery renders one gallery case's own template with its own
// theme and fixture, returning the HTML part Import.go then parses.
func renderGallery(t *testing.T, tc selfRoundTripCase) string {
	t.Helper()
	dir := filepath.Join("..", "examples", "gallery", tc.dir)
	set, err := gsxmail.Load(os.DirFS(dir), gsxmail.Options{Theme: tc.theme})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	props, err := tc.props()
	if err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}
	parts, err := set.Render(tc.template, props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return parts.HTML
}

// TestSelfRoundTripStructure is proof-bar item (a): render >= 3 (here,
// all five) shipped gallery templates, import each rendered HTML back,
// and assert the recovered block-kind sequence matches the source
// template's own declared component sequence exactly.
func TestSelfRoundTripStructure(t *testing.T) {
	for _, tc := range selfRoundTripCases() {
		t.Run(tc.dir, func(t *testing.T) {
			html := renderGallery(t, tc)
			res, err := Import([]byte(html), tc.dir+".html", Options{})
			if err != nil {
				t.Fatalf("Import: %v", err)
			}
			got := res.BlockKinds()
			if !equalStrings(got, tc.want) {
				t.Errorf("BlockKinds = %v, want %v\nunmapped:\n%s", got, tc.want, res.Report.Summary())
			}
		})
	}
}

// TestSelfRoundTripLoadsAndRenders is proof-bar item (c) applied to the
// self-round-trip corpus: every imported .gsx must load through
// gsxmail.Load with no error-severity finding, and render without error
// using its own harvested sample props.
func TestSelfRoundTripLoadsAndRenders(t *testing.T) {
	for _, tc := range selfRoundTripCases() {
		t.Run(tc.dir, func(t *testing.T) {
			html := renderGallery(t, tc)
			res, err := Import([]byte(html), tc.dir+".html", Options{})
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

// TestSelfRoundTripIdempotent is proof-bar item (d): import(render(
// import(x))) recovers the same block-kind sequence as import(x) — once
// a template has passed through gsxmail's own writer, re-importing its
// own output is stable.
func TestSelfRoundTripIdempotent(t *testing.T) {
	for _, tc := range selfRoundTripCases() {
		t.Run(tc.dir, func(t *testing.T) {
			html := renderGallery(t, tc)
			first, err := Import([]byte(html), tc.dir+".html", Options{})
			if err != nil {
				t.Fatalf("Import(x): %v", err)
			}
			_, renderedAgain := loadAndRenderHTML(t, first, first.TemplateName)
			second, err := Import([]byte(renderedAgain), tc.dir+".html", Options{})
			if err != nil {
				t.Fatalf("Import(render(import(x))): %v", err)
			}
			got, want := second.BlockKinds(), first.BlockKinds()
			if !equalStrings(got, want) {
				t.Errorf("import(render(import(x))) = %v, want %v (import(x))", got, want)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
