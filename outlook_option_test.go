package gsxmail_test

import (
	"errors"
	"os"
	"testing"

	"m31labs.dev/gsxmail"
)

// TestOptionsOutlookRejectsInvalidValues is the launch-gate M6 finding's
// own proof: Options.Outlook outside {"", "ghost-tables", "off"} now
// fails Load closed (EM195), instead of silently falling through
// WriteOptions.hardened's own "!= \"off\"" check and defaulting to
// hardened mode for a value that was never meant to select it — the
// exact shape probes-gsxmail/ol's own probe exercised ("OFF", "of",
// "gost-tables", "true").
func TestOptionsOutlookRejectsInvalidValues(t *testing.T) {
	for _, valid := range []string{"", "ghost-tables", "off"} {
		t.Run("valid/"+valid, func(t *testing.T) {
			if _, err := gsxmail.Load(os.DirFS("testdata/wp52/emails"), gsxmail.Options{Outlook: valid}); err != nil {
				t.Errorf("Load(Options{Outlook: %q}) failed: %v", valid, err)
			}
		})
	}

	for _, invalid := range []string{"OFF", "of", "gost-tables", "true", "ghost-table"} {
		t.Run("invalid/"+invalid, func(t *testing.T) {
			_, err := gsxmail.Load(os.DirFS("testdata/wp52/emails"), gsxmail.Options{Outlook: invalid})
			if err == nil {
				t.Fatalf("Load(Options{Outlook: %q}) succeeded; want a fail-closed EM195", invalid)
			}
			var lintErr *gsxmail.LintError
			if !errors.As(err, &lintErr) {
				t.Fatalf("Load's error is not a *gsxmail.LintError: %v", err)
			}
			want := `Options.Outlook must be "", "ghost-tables", or "off"; got "` + invalid + `"`
			var found bool
			for _, d := range lintErr.Diagnostics {
				if d.Code == "EM195" && d.Message == want {
					found = true
				}
			}
			if !found {
				t.Errorf("no EM195 diagnostic %q found; got:\n%s", want, diagnosticsList(lintErr.Diagnostics))
			}
		})
	}
}
