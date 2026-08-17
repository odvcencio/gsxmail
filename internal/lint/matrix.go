package lint

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

// snapshot.json is a trimmed, embedded copy of the caniemail dataset
// (https://www.caniemail.com/api/data.json), captured on the date recorded
// in its own "date" field. `gsxmail matrix refresh` rewrites it; nothing
// else does. See Client and PropertySupport for exactly which fields
// survive the trim, and BuildSnapshot for how.
//
//go:embed snapshot.json
var snapshotJSON []byte

// Client is one entry in the default supported-client set the embedded
// snapshot carries support data for: Gmail web/iOS/Android, Apple Mail
// macOS/iOS, Outlook Windows desktop/web, Yahoo web.
type Client struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// PropertySupport is one CSS property's per-client support codes from the
// embedded caniemail snapshot: "y" (supported), "a" (partial/mitigated),
// "n" (unsupported), or "u" (no data). Support is keyed by Client.ID.
type PropertySupport struct {
	Slug    string            `json:"slug"`
	Support map[string]string `json:"support"`
}

// Snapshot is the embedded, dated caniemail client-support dataset,
// trimmed to the CSS-property features and the client/platform pairs the
// default supported-client set needs.
//
// License and Attribution record the caniemail dataset's own terms: the
// maintained source (https://github.com/HTeuMeuLeu/caniemail) states
// "MIT Licence" in its
// own README, not the CC BY 4.0 this project once assumed — verified by
// fetching that README directly, not carried over from memory.
type Snapshot struct {
	Date        string                     `json:"date"`
	Source      string                     `json:"source"`
	License     string                     `json:"license"`
	Attribution string                     `json:"attribution"`
	Clients     []Client                   `json:"clients"`
	Properties  map[string]PropertySupport `json:"properties"`
}

// ParseSnapshot decodes b as a Snapshot and validates its shape (T7's
// self-check: the embedded snapshot must parse, must be dated, and must
// declare its client set). `gsxmail matrix refresh` calls it on freshly
// downloaded, freshly trimmed data before writing snapshot.json; T7 calls
// it on the embedded bytes to prove the shipped file itself still parses.
func ParseSnapshot(b []byte) (*Snapshot, error) {
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("gsxmail: parsing caniemail snapshot: %w", err)
	}
	if s.Date == "" {
		return nil, fmt.Errorf("gsxmail: caniemail snapshot has no date")
	}
	if s.License == "" {
		return nil, fmt.Errorf("gsxmail: caniemail snapshot has no license")
	}
	if len(s.Clients) == 0 {
		return nil, fmt.Errorf("gsxmail: caniemail snapshot declares no clients")
	}
	if len(s.Properties) == 0 {
		return nil, fmt.Errorf("gsxmail: caniemail snapshot declares no properties")
	}
	return &s, nil
}

var defaultSnapshot = func() *Snapshot {
	s, err := ParseSnapshot(snapshotJSON)
	if err != nil {
		panic("gsxmail: embedded caniemail snapshot: " + err.Error())
	}
	return s
}()

// EmbeddedSnapshotJSON returns the raw bytes of the snapshot this build
// embeds, unparsed. `gsxmail matrix refresh` uses it to print a diff
// against the freshly fetched data before overwriting the file.
func EmbeddedSnapshotJSON() []byte {
	out := make([]byte, len(snapshotJSON))
	copy(out, snapshotJSON)
	return out
}

// Matrix answers style-property support questions against one Snapshot.
type Matrix struct {
	snap *Snapshot
}

// DefaultMatrix wraps the snapshot embedded in this gsxmail build.
func DefaultMatrix() *Matrix {
	return &Matrix{snap: defaultSnapshot}
}

// NewMatrix wraps an explicit Snapshot: `gsxmail matrix refresh`'s own
// verification path, and tests.
func NewMatrix(s *Snapshot) *Matrix {
	return &Matrix{snap: s}
}

// SnapshotDate is the caniemail dataset date the wrapped Snapshot was
// captured from.
func (m *Matrix) SnapshotDate() string { return m.snap.Date }

// SnapshotAttribution is the caniemail dataset's own license and
// attribution line, for a caller that wants to
// surface it (the README's "Prior art and attribution" section carries
// the same text).
func (m *Matrix) SnapshotAttribution() string { return m.snap.Attribution }

// Clients lists the default supported-client set, in the snapshot's
// declared order.
func (m *Matrix) Clients() []Client {
	out := make([]Client, len(m.snap.Clients))
	copy(out, m.snap.Clients)
	return out
}

// Finding is one client's non-full support for a style property.
type Finding struct {
	ClientLabel string
	// Code is the raw caniemail support code: "n" (EM101, unsupported) or
	// "a" (EM102, partial).
	Code string
}

// Check looks up property (a CSS property name) and returns one Finding
// per client whose latest recorded support status is not "y": Code "n"
// selects EM101 (error), Code "a" selects EM102 (warn). An unknown
// property, or a client the snapshot has no data for ("u"), contributes no
// Finding — the lint has nothing to say about it.
func (m *Matrix) Check(property string) []Finding {
	ps, ok := m.snap.Properties[property]
	if !ok {
		return nil
	}
	var out []Finding
	for _, c := range m.snap.Clients {
		switch ps.Support[c.ID] {
		case "n", "a":
			out = append(out, Finding{ClientLabel: c.Label, Code: ps.Support[c.ID]})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ClientLabel < out[j].ClientLabel })
	return out
}

// SupportWord renders a raw caniemail support code as the short phrase
// EM101/EM102's messages quote.
func SupportWord(code string) string {
	switch code {
	case "y":
		return "supported"
	case "a":
		return "partially supported"
	case "n":
		return "not supported"
	default:
		return "unknown"
	}
}
