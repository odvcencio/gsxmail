package gsxmail_test

import (
	"errors"
	"os"
	"testing"

	"m31labs.dev/gsxmail"
)

// TestColumnImageRunsEM111EM112 is the launch-gate m4 finding's own
// proof: email.Column's imgSrc/imgAlt now run the same EM111 (absolute
// https src) and EM112 (non-empty alt) checks a raw <img> and
// email.Hero already run, but only when imgSrc is actually set — Column's
// image is optional, unlike Hero's required one.
func TestColumnImageRunsEM111EM112(t *testing.T) {
	_, err := gsxmail.Load(os.DirFS("testdata/lint/columnimage"), gsxmail.Options{})
	if err == nil {
		t.Fatal("Load succeeded; testdata/lint/columnimage's relative src and empty alt cases are supposed to fail closed")
	}
	var lintErr *gsxmail.LintError
	if !errors.As(err, &lintErr) {
		t.Fatalf("Load's error is not a *gsxmail.LintError: %v", err)
	}

	want := []struct{ code, message string }{
		{"EM111", `img src must be an absolute https URL; mail clients cannot resolve relative paths`},
		{"EM112", `img requires non-empty alt text; image blocking is a common default`},
	}
	for _, tc := range want {
		t.Run(tc.code, func(t *testing.T) {
			for _, d := range lintErr.Diagnostics {
				if d.Code == tc.code && d.Message == tc.message {
					return
				}
			}
			t.Errorf("no diagnostic %s: %q found; got:\n%s", tc.code, tc.message, diagnosticsList(lintErr.Diagnostics))
		})
	}

	// CaseNoImage's own two Columns (neither sets imgSrc) must not
	// contribute any EM111/EM112 finding at all — only three diagnostics
	// total should exist (EM111 and EM112 from CaseRelativeSrc/
	// CaseEmptyAlt's own single bad Column each, EM112 firing only once
	// per case since each fixture trips exactly one of the two rules).
	var em111, em112 int
	for _, d := range lintErr.Diagnostics {
		switch d.Code {
		case "EM111":
			em111++
		case "EM112":
			em112++
		}
	}
	if em111 != 1 {
		t.Errorf("got %d EM111 findings, want exactly 1 (CaseNoImage's image-free Columns must not trip it)", em111)
	}
	if em112 != 1 {
		t.Errorf("got %d EM112 findings, want exactly 1 (CaseNoImage's image-free Columns must not trip it)", em112)
	}
}
