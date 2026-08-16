// Package gallery_test renders every gallery template (design spec section
// 15, WP5.3; pixel dossier section 8.1) against its own fixture, pins the
// golden pair, and runs the structural verification pass over the whole
// gallery in both output contracts — the corpus this package's own README
// promises doubles as (task instructions, "Rules").
package gallery_test

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
	"m31labs.dev/gsxmail/internal/structverify"
)

// galleryCase is one template's full test wiring: where its .gsx source
// and fixture JSON live, its declared template name, the Theme its own
// gallery README recommends, and a fresh zero value of its declared props
// struct for json.Unmarshal to decode the fixture into.
type galleryCase struct {
	dir      string // relative to this file's own directory
	template string
	theme    gsxmail.Theme
	props    func() (any, error)
}

func loadFixture(dir, file string, out any) error {
	data, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func galleryCases() []galleryCase {
	return []galleryCase{
		{
			dir: "welcome", template: "WelcomeEmail", theme: gsxmail.DefaultTheme(),
			props: func() (any, error) {
				var p welcome.WelcomeProps
				err := loadFixture("welcome", "welcome.props.json", &p)
				return p, err
			},
		},
		{
			dir: "magiclink", template: "MagicLinkEmail", theme: gsxmail.DefaultTheme(),
			props: func() (any, error) {
				var p magiclink.MagicLinkProps
				err := loadFixture("magiclink", "magiclink.props.json", &p)
				return p, err
			},
		},
		{
			dir: "receipt", template: "ReceiptEmail", theme: gsxmail.DefaultTheme(),
			props: func() (any, error) {
				var p receipt.ReceiptProps
				err := loadFixture("receipt", "receipt.props.json", &p)
				return p, err
			},
		},
		{
			// Ledger is DarkMode "adaptive" (theme.go's own doc comment):
			// Digest is the gallery's adaptive-mode demonstration.
			dir: "digest", template: "DigestEmail", theme: gsxmail.LedgerTheme(),
			props: func() (any, error) {
				var p digest.DigestProps
				err := loadFixture("digest", "digest.props.json", &p)
				return p, err
			},
		},
		{
			// Terminal is DarkMode "locked": Alert is the gallery's
			// dark-native demonstration (pixel dossier section 8.1's own
			// "locked dark theme" fixture highlight for Alert).
			dir: "alert", template: "AlertEmail", theme: gsxmail.TerminalTheme(),
			props: func() (any, error) {
				var p alert.AlertProps
				err := loadFixture("alert", "alert.props.json", &p)
				return p, err
			},
		},
	}
}

// TestGalleryGoldens renders every template with its own Theme and
// fixture and pins the resulting HTML/text pair.
func TestGalleryGoldens(t *testing.T) {
	for _, tc := range galleryCases() {
		t.Run(tc.dir, func(t *testing.T) {
			set, err := gsxmail.Load(os.DirFS(tc.dir), gsxmail.Options{Theme: tc.theme})
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

			wantHTML, err := os.ReadFile(filepath.Join(tc.dir, tc.dir+".html"))
			if err != nil {
				t.Fatal(err)
			}
			wantText, err := os.ReadFile(filepath.Join(tc.dir, tc.dir+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			if parts.HTML != string(wantHTML) {
				t.Errorf("HTML part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.HTML, wantHTML)
			}
			if parts.Text != string(wantText) {
				t.Errorf("text part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.Text, wantText)
			}
		})
	}
}

// TestGalleryStructuralPass runs the structural verification pass over
// the whole gallery in both output contracts (hardened and parity) plus
// every template's own recommended Theme (task instructions, "Rules":
// "Structverify runs over the whole gallery in both contracts + adaptive
// mode" — Digest's own Ledger theme, DarkMode "adaptive", is what
// exercises the EM174 adaptive-style-layer checks here).
func TestGalleryStructuralPass(t *testing.T) {
	for _, tc := range galleryCases() {
		t.Run(tc.dir, func(t *testing.T) {
			props, err := tc.props()
			if err != nil {
				t.Fatalf("decoding fixture: %v", err)
			}

			hardSet, err := gsxmail.Load(os.DirFS(tc.dir), gsxmail.Options{Theme: tc.theme})
			if err != nil {
				t.Fatalf("Load (hardened): %v", err)
			}
			hardParts, err := hardSet.Render(tc.template, props)
			if err != nil {
				t.Fatalf("Render (hardened): %v", err)
			}
			findings, err := structverify.VerifyContract(hardParts.HTML)
			if err != nil {
				t.Fatalf("VerifyContract (hardened): %v", err)
			}
			if len(findings) > 0 {
				t.Errorf("%s (hardened) fails the structural pass:\n%s", tc.template, joinFindings(findings))
			}

			paritySet, err := gsxmail.Load(os.DirFS(tc.dir), gsxmail.Options{Theme: tc.theme, Outlook: "off"})
			if err != nil {
				t.Fatalf("Load (parity): %v", err)
			}
			parityParts, err := paritySet.Render(tc.template, props)
			if err != nil {
				t.Fatalf("Render (parity): %v", err)
			}
			findings, err = structverify.Verify(parityParts.HTML)
			if err != nil {
				t.Fatalf("Verify (parity): %v", err)
			}
			if len(findings) > 0 {
				t.Errorf("%s (parity) is not well-formed:\n%s", tc.template, joinFindings(findings))
			}
		})
	}
}

// TestGalleryUnderBudget proves every gallery HTML stays under the
// EM121 90KB warning line (task instructions, "Acceptance": "every
// gallery HTML under the EM121 90 KB warning line").
func TestGalleryUnderBudget(t *testing.T) {
	const warnBytes = 90_000
	for _, tc := range galleryCases() {
		t.Run(tc.dir, func(t *testing.T) {
			set, err := gsxmail.Load(os.DirFS(tc.dir), gsxmail.Options{Theme: tc.theme})
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
			if len(parts.HTML) >= warnBytes {
				t.Errorf("%s HTML is %d bytes; the EM121 warning line is %d", tc.template, len(parts.HTML), warnBytes)
			}
		})
	}
}

func joinFindings(findings []structverify.Finding) string {
	s := ""
	for _, f := range findings {
		s += "  " + f.String() + "\n"
	}
	return s
}
