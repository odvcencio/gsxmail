package lint

import "testing"

// gracefulDegradations lists the small, deliberate exceptions to "every
// property the stdlib emits is y or a across the supported-client set"
// (spec section 11, T7): properties where the real caniemail dataset
// reports full non-support ("n") for one client, but the gap is a known,
// industry-accepted, visually safe fallback — not a bug this lint should
// block the stdlib on.
//
//   - border-radius / Outlook (Windows desktop): the Word rendering engine
//     ignores border-radius outright and falls back to square corners.
//     Every comparable framework (MJML, react-email, Maizzle) emits it
//     anyway; gsxmail's own theme (theme.go) does too.
//   - height / Yahoo (web): caniemail's height feature specifically tests
//     height on a <div>, footnoted as such; the stdlib only ever sets
//     height on a <td> (renderhtml's badge cell), the well-documented
//     email-safe pattern the div case does not cover.
var gracefulDegradations = map[string]map[string]bool{
	"border-radius": {"outlook-windows": true},
	"height":        {"yahoo-web": true},
}

// stdlibProperties lists every CSS property renderhtml's writer emits
// (theme.go's tokens fill in the values, never the property names, so this
// list is exhaustive and stable across themes).
var stdlibProperties = []string{
	"margin", "padding", "background-color", "width", "max-width",
	"border", "border-radius", "border-bottom", "border-top",
	"color", "font-family", "font-size", "font-weight",
	"letter-spacing", "text-transform", "margin-top", "margin-bottom",
	"line-height", "text-decoration", "display", "vertical-align",
	"text-align", "height", "border-collapse",
}

// TestEmbeddedSnapshotParses is T7's first half: the embedded snapshot
// parses and carries a date.
func TestEmbeddedSnapshotParses(t *testing.T) {
	snap, err := ParseSnapshot(EmbeddedSnapshotJSON())
	if err != nil {
		t.Fatalf("embedded snapshot.json does not parse: %v", err)
	}
	if snap.Date == "" {
		t.Error("embedded snapshot has no date")
	}
}

// TestSnapshotCoversDefaultClientSet is T7's client-set half: the embedded
// snapshot declares support data for every client in the design spec's
// proposed default set (section 16, open question 3).
func TestSnapshotCoversDefaultClientSet(t *testing.T) {
	want := []string{
		"gmail-web", "gmail-ios", "gmail-android",
		"apple-mail-macos", "apple-mail-ios",
		"outlook-windows", "outlook-web",
		"yahoo-web",
	}
	m := DefaultMatrix()
	got := map[string]bool{}
	for _, c := range m.Clients() {
		got[c.ID] = true
	}
	for _, id := range want {
		if !got[id] {
			t.Errorf("embedded snapshot's client set is missing %s", id)
		}
	}
	if len(m.Clients()) != len(want) {
		t.Errorf("embedded snapshot declares %d clients, want exactly %d (%v)", len(m.Clients()), len(want), want)
	}
}

// TestStdlibNeverViolatesItsOwnLint is T7's third half (spec section 11:
// "Stdlib never violates its own lint"): every CSS property the stdlib
// writer emits is "y" or "a" — never "n" — across the default
// supported-client set, with the two documented, graceful exceptions above.
func TestStdlibNeverViolatesItsOwnLint(t *testing.T) {
	m := DefaultMatrix()
	for _, prop := range stdlibProperties {
		for _, f := range m.Check(prop) {
			if f.Code != "n" {
				continue // "a" (partial) is allowed; T7 only forbids "n".
			}
			exceptClient := gracefulDegradations[prop]
			if exceptClient != nil && clientIDForLabel(m, f.ClientLabel) != "" && exceptClient[clientIDForLabel(m, f.ClientLabel)] {
				continue
			}
			t.Errorf("stdlib property %q is unsupported (n) in %s; the stdlib's own lint would reject this style — either stop emitting it or add it to gracefulDegradations with a reason", prop, f.ClientLabel)
		}
	}
}

func clientIDForLabel(m *Matrix, label string) string {
	for _, c := range m.Clients() {
		if c.Label == label {
			return c.ID
		}
	}
	return ""
}
