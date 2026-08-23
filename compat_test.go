package gsxmail_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gsxmail"
)

// gsxLane returns which compatibility lane this suite is running in, as
// declared by the environment: "pinned" (the default, and what CI's
// pinned matrix job sets), "latest" (CI's latest matrix job, which bumps
// m31labs.dev/gosx to the newest tagged release before building), or ""
// for an ordinary local run. Anything else is a typo in the workflow and
// fails loudly rather than silently changing what the lane proves.
func gsxLane(t *testing.T) string {
	t.Helper()
	switch lane := os.Getenv("GSXMAIL_GOSX_LANE"); lane {
	case "", "pinned", "latest":
		return lane
	default:
		t.Fatalf("GSXMAIL_GOSX_LANE = %q; want \"pinned\", \"latest\", or unset", lane)
		return ""
	}
}

// goModGosxRequire returns the m31labs.dev/gosx version this module's own
// go.mod names — the release this checkout is pinned to and the one every
// checked-in golden is verified against.
func goModGosxRequire(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^\s*m31labs\.dev/gosx v(\S+)`)
	m := re.FindSubmatch(data)
	if m == nil {
		t.Fatal("go.mod has no m31labs.dev/gosx require line")
	}
	return string(m[1])
}

// readmeGosxPin returns the m31labs.dev/gosx version README's "gosx
// version window" section names — the support statement users read.
func readmeGosxPin(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	section := regexp.MustCompile(`(?s)## gosx version window.*?(?:^## |\z)`).Find(data)
	if section == nil {
		t.Fatal("README.md has no \"gosx version window\" section")
	}
	re := regexp.MustCompile("m31labs\\.dev/gosx v(\\S+?)`")
	m := re.FindSubmatch(section)
	if m == nil {
		t.Fatal("README's gosx version window does not name a `m31labs.dev/gosx vX.Y.Z` pin")
	}
	return string(m[1])
}

// TestLinkedGosxMatchesGoMod ties the suite's verdict to a named release:
// the gosx build linked into this test binary must be the one go.mod
// requires. In the pinned lane that equality is what makes a green run
// mean "this exact release is verified". A build moved off go.mod — CI's
// latest lane bumps the require line before building, and a downstream
// build graph can move the linked release through minimal version
// selection — cannot have its verdict attached to the checkout's pin, so
// outside the pinned lane the test skips rather than failing on a version
// literal: a moved build is proven by the rest of this suite passing
// against it, not by a string comparison.
func TestLinkedGosxMatchesGoMod(t *testing.T) {
	require := goModGosxRequire(t)
	if gosx.Version == require {
		t.Logf("linked gosx v%s == go.mod's require line: a green run verifies v%s", gosx.Version, gosx.Version)
		return
	}
	if lane := gsxLane(t); lane == "pinned" {
		t.Fatalf("pinned lane links gosx v%s but go.mod requires v%s: the lane's verdict would not describe the checkout's pin", gosx.Version, require)
	}
	t.Skipf("linked gosx v%s != go.mod's v%s: this build was moved off the checkout's pin; the suite's verdict applies to v%s", gosx.Version, require, gosx.Version)
}

// TestReadmeNamesThePinnedGosx keeps README's "gosx version window"
// section truthful: outside the latest lane, it must name exactly the
// release go.mod requires, because the section is where users read what
// gsxmail is tested against. The latest lane skips — its go.mod was
// bumped on purpose, and the README correctly keeps describing the
// development pin until the bump itself lands as a commit.
func TestReadmeNamesThePinnedGosx(t *testing.T) {
	if gsxLane(t) == "latest" {
		t.Skip("latest lane: go.mod was bumped past the development pin; the README keeps describing the pin until the bump lands")
	}
	require := goModGosxRequire(t)
	pin := readmeGosxPin(t)
	if pin != require {
		t.Errorf("README's gosx version window names v%s but go.mod requires v%s; update the section when the pin moves", pin, require)
	}
}

// TestLatestLaneMovesOffThePin keeps CI's latest lane honest: a run that
// claims to test the newest tagged gosx release must actually link a
// different release than the pin, or the lane silently duplicates the
// pinned lane and the compatibility window goes unproven while looking
// covered. The bump resolving to the pin itself (no newer release exists)
// is not a failure — the pinned lane already covers that exact build — so
// it skips with the situation spelled out in the log.
func TestLatestLaneMovesOffThePin(t *testing.T) {
	if gsxLane(t) != "latest" {
		t.Skip("not the latest lane")
	}
	pin := readmeGosxPin(t)
	if gosx.Version == pin {
		t.Skipf("GSXMAIL_GOSX_LANE=latest but this build linked the pinned v%s: the bump resolved to the pin, so this lane proves nothing the pinned lane does not already cover", pin)
	}
	t.Logf("latest lane is genuinely exercising gosx v%s against the pinned v%s", gosx.Version, pin)
}

// TestLoadDiagnosticsCarryNoVersionSkew pins the contract that repaired
// the false-red compatibility lane: diagnostics describe templates, not
// the build. A known-clean template set must Check clean whatever gosx
// release this binary linked — no version-skew finding (the retired
// EM194), and no diagnostic message embedding a gosx version literal.
func TestLoadDiagnosticsCarryNoVersionSkew(t *testing.T) {
	set, err := gsxmail.Load(os.DirFS("examples/quickstart/emails"), gsxmail.Options{Dir: "examples/quickstart/emails"})
	if err != nil {
		t.Fatalf("Load(quickstart): %v", err)
	}
	diags := set.Check()
	if len(diags) != 0 {
		t.Errorf("Check() on the clean quickstart fixture = %v, want none whatever gosx release is linked (this build: v%s)", diags, gosx.Version)
	}
	for _, d := range diags {
		if strings.Contains(d.Message, "m31labs.dev/gosx") || strings.Contains(d.Message, gosx.Version) {
			t.Errorf("diagnostic %s embeds gosx version state; template diagnostics must not encode build or version information", d.String())
		}
	}
}

// TestGosxVersionIsBareSemver checks the one version-format assumption the
// tests above and the docs rely on: gosx.Version is the release's bare
// semver (no leading "v"), the same form go.mod's require line and README's
// pin use, so the comparisons compare releases and not spellings.
func TestGosxVersionIsBareSemver(t *testing.T) {
	// A leading "v" would make every comparison against go.mod's and the
	// README's bare-semver spellings silently never match.
	if !regexp.MustCompile(`^\d+\.\d+\.\S+$`).MatchString(gosx.Version) {
		t.Errorf("gosx.Version = %q, want bare semver without a leading %q", gosx.Version, "v")
	}
}
