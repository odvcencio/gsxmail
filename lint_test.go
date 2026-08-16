package gsxmail_test

import (
	"errors"
	"os"
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

func diagnosticsList(diags []gsxmail.Diagnostic) string {
	s := ""
	for _, d := range diags {
		s += "  " + d.String() + "\n"
	}
	return s
}
