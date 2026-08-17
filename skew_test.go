package gsxmail

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

// TestGosxPinnedVersionMatchesGoMod pins gosxPinnedVersion to go.mod's own
// require line, mirroring gosx's own internal/version package
// (TestNumberMatchesCurrent's same reasoning): nothing else enforces that
// the two agree, so a go.mod bump that forgets to update the constant
// would make checkGosxSkew warn on every Load of an up-to-date build.
func TestGosxPinnedVersionMatchesGoMod(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`m31labs\.dev/gosx v(\S+)`)
	m := re.FindSubmatch(data)
	if m == nil {
		t.Fatal("go.mod has no m31labs.dev/gosx require line")
	}
	want := string(m[1])
	if gosxPinnedVersion != want {
		t.Errorf("gosxPinnedVersion = %q, go.mod requires v%s; update skew.go's constant", gosxPinnedVersion, want)
	}
}

// TestGosxSkewDiagnostic is the pure-decision test for
// gosxSkewDiagnostic: a match returns nil, and a mismatch returns exactly
// one warn-severity EM194 naming both versions.
func TestGosxSkewDiagnostic(t *testing.T) {
	if diags := gosxSkewDiagnostic("0.42.2", "0.42.2"); diags != nil {
		t.Errorf("gosxSkewDiagnostic(match) = %v, want nil", diags)
	}
	diags := gosxSkewDiagnostic("0.42.2", "0.43.0")
	if len(diags) != 1 {
		t.Fatalf("gosxSkewDiagnostic(mismatch) returned %d diagnostics, want 1", len(diags))
	}
	if diags[0].Code != "EM194" || diags[0].Severity != "warn" {
		t.Errorf("gosxSkewDiagnostic(mismatch) = %+v, want code EM194, severity warn", diags[0])
	}
	if !strings.Contains(diags[0].Message, "0.42.2") || !strings.Contains(diags[0].Message, "0.43.0") {
		t.Errorf("EM194 message %q does not name both versions", diags[0].Message)
	}
}

// TestCheckGosxSkewMatchesLinkedBuild proves checkGosxSkew itself reads
// the real gosx.Version this test binary links, not a stale copy: it is
// nil exactly when that version equals gosxPinnedVersion.
func TestCheckGosxSkewMatchesLinkedBuild(t *testing.T) {
	diags := checkGosxSkew()
	if gosx.Version == gosxPinnedVersion {
		if diags != nil {
			t.Errorf("checkGosxSkew() = %v, want nil when the linked gosx build (v%s) matches the pin", diags, gosx.Version)
		}
		return
	}
	if diags == nil {
		t.Errorf("checkGosxSkew() = nil, want an EM194 diagnostic: linked gosx v%s != pinned v%s", gosx.Version, gosxPinnedVersion)
	}
}
