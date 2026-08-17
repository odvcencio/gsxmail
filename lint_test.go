package gsxmail_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
)

// T5: the lint catalog's message strings, pinned verbatim (spec section 11,
// section 8). testdata/lint/catalog holds one minimal function per EM rule,
// each shaped to trigger exactly that rule; this test loads the whole
// directory once and asserts every expected (Code, Message) pair appears
// somewhere in the resulting *gsxmail.LintError. Options.Helpers registers
// "wrongArity" with two string parameters so CaseEM015's one-argument call
// mismatches on purpose; "undeclaredHelper" (CaseEM014) is deliberately
// left unregistered.
func TestLintCatalog(t *testing.T) {
	opts := gsxmail.Options{
		Helpers: map[string]any{
			"wrongArity": func(a, b string) string { return a + b },
		},
	}
	_, err := gsxmail.Load(os.DirFS("testdata/lint/catalog"), opts)
	if err == nil {
		t.Fatal("Load succeeded; every fixture in testdata/lint/catalog is supposed to fail closed")
	}
	var lintErr *gsxmail.LintError
	if !errors.As(err, &lintErr) {
		t.Fatalf("Load's error is not a *gsxmail.LintError: %v", err)
	}

	cases := []struct {
		code    string
		message string
	}{
		{"EM001", `element <script> is not allowed in an email template; mail clients do not run JavaScript`},
		{"EM002", `element <form> is not allowed in an email template; render an <email.CTA> link instead`},
		{"EM003", `element <blink> is not on the email element allowlist`},
		{"EM004", `attribute onclick is an event handler; mail clients strip JavaScript`},
		{"EM005", `directive //gosx:island has no meaning in an email template; emails have no client runtime`},
		{"EM006", `<link rel="stylesheet"> is not allowed; email styles must be inline or in the shell <style> block`},
		{"EM010", `expression "props.Name * 2" is not in the email dialect; allowed: props paths, loop bindings, literals, string +, comparisons, len(), registered helpers`},
		{"EM011", `expression "props.Items.length" reads .length; this is not Go — use len(props.Items)`},
		{"EM012", `template CaseEM012 reads props.Bogus but type Props has no field Bogus`},
		{"EM013", `props.Roster has type Roster; interpolated values must be string, integer, float, or bool — format structs before rendering`},
		{"EM014", `helper undeclaredHelper is not registered in Options.Helpers`},
		{"EM015", `helper wrongArity takes 2 arguments; the template passes 1`},
		{"EM020", `component <Something> is not an email.* component and is not declared in this template set`},
		{"EM030", `<If> requires exactly one cond attribute with a bool expression`},
		{"EM031", `<If> child is bare text; wrap it in an element so the text twin can place it`},
		{"EM032", `<Each of={props.DraftTime}> requires a slice or array props path; got string`},
		{"EM033", `<Each> requires as="name"; the binding name is the loop variable`},
		{"EM101", `style property "border-radius" is unsupported in Outlook (Windows desktop) (caniemail not supported, snapshot 2026-08-10); remove it or suppress with gsxmail:allow "border-radius"`},
		{"EM102", `style property "margin-top" has partial support in Gmail (Android) (caniemail partially supported); verify in preview`},
		{"EM103", `style attribute "not a declaration" does not parse as CSS declarations`},
		{"EM104", `attribute class is not supported on custom elements in v1; use style or an email.* component`},
		{"EM110", `href scheme "javascript" is not allowed; use https, http, or mailto`},
		{"EM111", `img src must be an absolute https URL; mail clients cannot resolve relative paths`},
		{"EM112", `img requires non-empty alt text; image blocking is a common default`},
		{"EM170", `Shell has no preheader; clients will preview the first body text instead`},
		{"EM171", `preheader is 157 characters; the limit is 150 (the emitted block pads to exactly 150)`},
		{"EM172", `<email.Shell> outlook attribute must be a static "", "ghost-tables", or "off"; got "sideways"`},
		{"EM175", `<email.Button> variant attribute must be a static "primary", "secondary", or "link"; got "diagonal"`},
		{"EM176", `<email.Columns> must contain between 2 and 4 <email.Column> children, got 1`},
		{"EM176", `<email.Columns> children must be <email.Column>, got "email.Signal"`},
		{"EM177", `<email.Column> must be a direct child of <email.Columns>`},
		{"EM178", `<email.Hero> requires width and height attributes at display size; retina assets render at intrinsic size without them`},
		{"EM179", `<email.Spacer> requires a height attribute (a positive pixel integer)`},
		{"EM180", `<email.Badge> tone attribute must be a static "neutral", "positive", "warning", or "critical"; got "sideways"`},
		{"EM181", `email.Spacer height must be a positive decimal integer (a pixel count), got "20px"`},
		{"EM190", `<email.Signal> has no "bogus" attribute`},
		{"EM191", `<email.CTA> requires a "href" attribute`},
		{"EM196", `<email.Bogus> is not on the email.* component list (Shell, Signal, Headline, Panel/PanelRow, CTA, Button, Columns/Column, Hero, Spacer, Badge, PickList/Item, Footer, Note, Divider, StatTable/StatRow)`},
		{"EM197", `<email.Panel> children must be <email.PanelRow>, got "email.Signal"`},
		{"EM198", `<email.PickList> children must be <email.Item>, got "email.Signal"`},
		{"EM199", `<email.StatTable> children must be <email.StatRow> or <Each>, got "email.Signal"`},
		{"EM199", `<Each> inside <email.StatTable> must wrap <email.StatRow>, got "email.Signal"`},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			for _, d := range lintErr.Diagnostics {
				if d.Code == tc.code && d.Message == tc.message {
					return
				}
			}
			t.Errorf("no diagnostic %s: %q found; got:\n%s", tc.code, tc.message, diagnosticsList(lintErr.Diagnostics))
		})
	}
}

// TestCustomSubtreeLintReachesRawStyles is WP3's regression check for the
// Custom pass-through (design spec section 7.2, section 15 WP3): the
// per-property matrix lint (EM101/EM102) must reach a raw element's style
// attribute wherever it sits, including nested inside <email.Shell> —
// the exact position lower.Lower now accepts as a Custom subtree, rather
// than rejecting outright.
func TestCustomSubtreeLintReachesRawStyles(t *testing.T) {
	_, err := gsxmail.Load(os.DirFS("testdata/lint/customsubtree"), gsxmail.Options{})
	if err == nil {
		t.Fatal("Load succeeded; the Custom subtree's border-radius should fail EM101")
	}
	var lintErr *gsxmail.LintError
	if !errors.As(err, &lintErr) {
		t.Fatalf("Load's error is not a *gsxmail.LintError: %v", err)
	}
	want := `style property "border-radius" is unsupported in Outlook (Windows desktop) (caniemail not supported, snapshot 2026-08-10); remove it or suppress with gsxmail:allow "border-radius"`
	for _, d := range lintErr.Diagnostics {
		if d.Code == "EM101" && d.Message == want {
			return
		}
	}
	t.Errorf("no EM101 diagnostic %q found; got:\n%s", want, diagnosticsList(lintErr.Diagnostics))
}

// TestEM143CustomBlockColor is WP5.2's regression check for
// checkDarkPaletteCoverage (design spec section 15, WP5.2; pixel dossier
// section 5.3, rule 3): a Custom element's literal color that matches
// neither the light Theme's nor Theme.Dark's own tokens warns, but only
// under DarkMode "adaptive" — the same fixture loaded under DefaultTheme
// (DarkMode "none") must not report it at all.
func TestEM143CustomBlockColor(t *testing.T) {
	adaptive := gsxmail.DefaultTheme()
	adaptive.DarkMode = "adaptive"
	adaptive.Dark = &gsxmail.DarkPalette{
		ColorGround: "#0A0A0D", ColorCard: "#101014", ColorPanel: "#17171C",
		ColorBorder: "#2A2A33", ColorAccent: "#6E8CFF", ColorInk: "#F2F2F5",
		ColorBody: "#D6D6DE", ColorMuted: "#9C9CA8", ColorFaint: "#6B6B76",
	}
	set, err := gsxmail.Load(os.DirFS("testdata/lint/darkcustom"), gsxmail.Options{Theme: adaptive})
	if err != nil {
		t.Fatalf("Load (adaptive): %v", err)
	}
	want := `Custom block color "#123456" has no dark-palette counterpart; forced transforms will recolor it unpredictably`
	var found bool
	for _, d := range set.Check() {
		if d.Code == "EM143" && d.Message == want {
			found = true
		}
	}
	if !found {
		t.Errorf("no EM143 diagnostic %q found; got:\n%s", want, diagnosticsList(set.Check()))
	}

	noneSet, err := gsxmail.Load(os.DirFS("testdata/lint/darkcustom"), gsxmail.Options{})
	if err != nil {
		t.Fatalf("Load (none): %v", err)
	}
	for _, d := range noneSet.Check() {
		if d.Code == "EM143" {
			t.Errorf("DarkMode \"none\" reported EM143, want it to run only under \"adaptive\": %v", d)
		}
	}
}

// TestNewcomerFixtureReportsEveryMistake reproduces probes-gsxmail/
// newcomer/brk's own template verbatim (testdata/lint/newcomer): the
// launch-gate B4 finding's four attribute mistakes a newcomer's first
// template made — heading= for title=, a CTA missing href entirely, and a
// Button whose label/href are spelled Label/HREF. Before B4's schema-based
// EM190/EM191 checks, gsxmail check passed this template with zero
// diagnostics. It must now report all four.
func TestNewcomerFixtureReportsEveryMistake(t *testing.T) {
	_, err := gsxmail.Load(os.DirFS("testdata/lint/newcomer"), gsxmail.Options{})
	if err == nil {
		t.Fatal("Load succeeded; every mistake in testdata/lint/newcomer is supposed to fail closed")
	}
	var lintErr *gsxmail.LintError
	if !errors.As(err, &lintErr) {
		t.Fatalf("Load's error is not a *gsxmail.LintError: %v", err)
	}

	want := []struct{ code, message string }{
		{"EM190", `<email.Headline> has no "heading" attribute`},
		{"EM191", `<email.Headline> requires a "title" attribute`},
		{"EM191", `<email.CTA> requires a "href" attribute`},
		{"EM190", `<email.Button> has no "Label" attribute`},
		{"EM190", `<email.Button> has no "HREF" attribute`},
		{"EM191", `<email.Button> requires a "label" attribute`},
		{"EM191", `<email.Button> requires a "href" attribute`},
	}
	for _, tc := range want {
		t.Run(tc.code+"_"+tc.message, func(t *testing.T) {
			for _, d := range lintErr.Diagnostics {
				if d.Code == tc.code && d.Message == tc.message {
					return
				}
			}
			t.Errorf("no diagnostic %s: %q found; got:\n%s", tc.code, tc.message, diagnosticsList(lintErr.Diagnostics))
		})
	}
}

// TestEM192PropsResolutionFailureSurfaces is the launch-gate B3, point 1
// regression check: a props type whose own package fails to type-check
// (here, because of an import path nothing can resolve) must report
// EM192, with the real cause in its message, and must not also flood the
// diagnostics list with a misleading EM012 "no such field" for every
// props.field read in the template — checkComponent's own propsUnresolved
// short-circuit.
func TestEM192PropsResolutionFailureSurfaces(t *testing.T) {
	_, err := gsxmail.Load(os.DirFS("testdata/lint/em192"), gsxmail.Options{})
	if err == nil {
		t.Fatal("Load succeeded; testdata/lint/em192's props package cannot type-check")
	}
	var lintErr *gsxmail.LintError
	if !errors.As(err, &lintErr) {
		t.Fatalf("Load's error is not a *gsxmail.LintError: %v", err)
	}

	var em192 *gsxmail.Diagnostic
	for i, d := range lintErr.Diagnostics {
		if d.Code == "EM012" {
			t.Errorf("got a misleading EM012 alongside the resolution failure: %s", d.String())
		}
		if d.Code == "EM192" {
			em192 = &lintErr.Diagnostics[i]
		}
	}
	if em192 == nil {
		t.Fatalf("no EM192 diagnostic found; got:\n%s", diagnosticsList(lintErr.Diagnostics))
	}
	if !strings.Contains(em192.Message, "props type BadProps could not be resolved") {
		t.Errorf("EM192 message = %q, want it to name the props type and say it could not be resolved", em192.Message)
	}
}

func diagnosticsList(diags []gsxmail.Diagnostic) string {
	s := ""
	for _, d := range diags {
		s += "  " + d.String() + "\n"
	}
	return s
}
