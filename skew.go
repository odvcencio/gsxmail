package gsxmail

import (
	"fmt"

	"m31labs.dev/gosx"
)

// gosxPinnedVersion is the m31labs.dev/gosx release (gosx.Version's own
// bare-semver form, no leading "v") gsxmail's own test suite and every
// checked-in golden are verified against — the same version go.mod's
// require line names. Go's module resolution can still link in a
// different one: another dependency in the same build graph may require
// a newer gosx release than gsxmail's own go.mod does, and minimal
// version selection picks the higher one for everybody. CI's own gosx
// {pinned, latest} matrix (see .github/workflows/ci.yml) is what actually
// proves gsxmail's behavior across that window; checkGosxSkew (following
// cmd/gosx's own checkVersionSkew) is Load's own early, in-process signal
// that the window may have moved.
const gosxPinnedVersion = "0.42.2"

// checkGosxSkew compares gosx.Version — the release actually linked into
// this build — against gosxPinnedVersion and returns a warning-severity
// Diagnostic when they differ, or nil when they match. Unlike cmd/gosx's
// own checkVersionSkew, this never fails Load closed: gsxmail is a
// library many processes import, not a single invoked CLI, so a resolved
// version outside gsxmail's own pin is a signal to investigate, not by
// itself proof that anything is actually broken — Load already ran the
// full lint catalog and lower pass against whatever gosx build this is;
// if those passed, the skew alone is not grounds to block a caller's own
// Load.
func checkGosxSkew() []Diagnostic {
	return gosxSkewDiagnostic(gosxPinnedVersion, gosx.Version)
}

// gosxSkewDiagnostic is checkGosxSkew's pure decision, kept separate from
// gosx.Version's own package-level access so it is testable without
// depending on which gosx release the test binary itself happens to link
// — the same separation cmd/gosx's own interpretGoListModule/
// versionSkewError split uses.
func gosxSkewDiagnostic(pinned, linked string) []Diagnostic {
	if pinned == linked {
		return nil
	}
	return []Diagnostic{{
		Code: "EM194", Severity: "warn",
		Message: fmt.Sprintf(
			"gsxmail is developed and tested against m31labs.dev/gosx v%s; this build resolved v%s instead — if you see unexpected check or render differences, pin gosx back to v%s or file an issue",
			pinned, linked, pinned),
	}}
}
