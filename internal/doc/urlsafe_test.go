package doc

import (
	"reflect"
	"testing"
)

// TestSafeURLAllowlist pins SafeURL's own scheme policy — the single
// EM110 allowlist both writers share (renderhtml/href.go and
// rendertext/rendertext.go each delegate here). https, http, and mailto
// pass; every other scheme (javascript:, data:, vbscript:, file:, ftp:),
// a scheme-less value (relative path, fragment, protocol-relative), and
// anything url.Parse rejects must fail closed. Case-insensitivity on the
// scheme ("JAVASCRIPT:" must not slip past a ToLower-less comparison) and
// whitespace padding (a props value with stray spaces still parses) are
// part of the contract, not incidental.
func TestSafeURLAllowlist(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"https://example.com/x", true},
		{"http://example.com", true},
		{"mailto:test@example.com", true},
		{"HTTPS://EXAMPLE.COM/UPPER", true},
		{"MAILTO:test@example.com", true},
		{"https://example.com/a?b=c#d", true},
		{"  https://example.com/padded  ", true},

		{"javascript:alert(document.cookie)", false},
		{"JAVASCRIPT:alert(1)", false},
		{"data:text/html,<script>alert(1)</script>", false},
		{"vbscript:MsgBox", false},
		{"file:///etc/passwd", false},
		{"ftp://example.com/pub", false},
		{"#fragment-only", false},
		{"/relative/path", false},
		{"relative/path.png", false},
		{"//cdn.example/protocol-relative.png", false},
		{"", false},
		{"   ", false},
		{"http://[::1", false}, // malformed: url.Parse itself errors
	}
	for _, tc := range cases {
		if got := SafeURL(tc.raw); got != tc.want {
			t.Errorf("SafeURL(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

// TestSafeImgSrcAllowlist pins SafeImgSrc's own EM111 rule: an absolute
// https URL, nothing else. http would leak the recipient's request over
// plaintext; relative and protocol-relative paths cannot resolve inside a
// mail client; data: and script schemes are never an image origin.
func TestSafeImgSrcAllowlist(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"https://cdn.example/i.png", true},
		{"  https://cdn.example/padded.png  ", true},
		{"http://cdn.example/insecure.png", false},
		{"//cdn.example/protocol-relative.png", false},
		{"assets/relative.png", false},
		{"data:image/png;base64,iVBORw0KGgo=", false},
		{"javascript:alert(1)", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := SafeImgSrc(tc.raw); got != tc.want {
			t.Errorf("SafeImgSrc(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

// auditFixture is one Resolved tree carrying every site kind AuditURLs
// walks, in block order: CTA href, Button href, Hero src, two Columns
// (one image-free, one unsafe, one safe), and a Custom subtree whose
// unsafe sites sit at three different depths (including a nested <a> and
// an <img> with no src attribute at all).
func auditFixture() *Resolved {
	return &Resolved{
		Shell: ResolvedShell{Title: "audit fixture", Lang: "en"},
		Blocks: []ResolvedBlock{
			ResolvedCTA{Label: "Act", Href: "javascript:alert(document.cookie)"},
			ResolvedButton{Variant: "link", Label: "Data", Href: "data:text/html,x"},
			ResolvedHero{Src: "http://cdn.example/hero.png", Alt: "hero alt"},
			ResolvedColumns{Columns: []ResolvedColumn{
				{Title: "no image on this one"},               // image-free: nothing to judge
				{ImgSrc: "assets/column.png", ImgAlt: "col"},  // relative: reported
				{ImgSrc: "https://cdn.example/ok-column.png"}, // safe: silent
			}},
			ResolvedCustom{Root: ResolvedCustomNode{
				Tag: "div",
				Children: []ResolvedCustomNode{
					{Tag: "a",
						Attrs:    []ResolvedCustomAttr{{Name: "href", Value: "vbscript:x"}},
						Children: []ResolvedCustomNode{{IsText: true, Text: "vbs link"}}},
					{Tag: "img",
						Attrs: []ResolvedCustomAttr{{Name: "src", Value: "//cdn.example/pr.png"}, {Name: "alt", Value: "pr"}}},
					{Tag: "p", Children: []ResolvedCustomNode{
						{Tag: "a",
							Attrs:    []ResolvedCustomAttr{{Name: "href", Value: "/deeply/nested/relative"}},
							Children: []ResolvedCustomNode{{IsText: true, Text: "deep rel"}}},
						{Tag: "img",
							Attrs: []ResolvedCustomAttr{{Name: "alt", Value: "no src attribute"}}},
					}},
				},
			}},
		},
	}
}

// wantAuditSites is the complete, ordered set auditFixture's tree must
// produce: one URLSite per failing href/img src, blocks in order, Custom
// subtrees in depth-first source order. The image-free Column, the safe
// Column image, the Custom img with no src attribute, and every safe href
// contribute nothing.
var wantAuditSites = []URLSite{
	{Component: "email.CTA", Field: "href", Value: "javascript:alert(document.cookie)"},
	{Component: "email.Button", Field: "href", Value: "data:text/html,x"},
	{Component: "email.Hero", Field: "img src", Value: "http://cdn.example/hero.png"},
	{Component: "email.Column", Field: "img src", Value: "assets/column.png"},
	{Component: "email.Custom", Field: "href", Value: "vbscript:x"},
	{Component: "email.Custom", Field: "img src", Value: "//cdn.example/pr.png"},
	{Component: "email.Custom", Field: "href", Value: "/deeply/nested/relative"},
}

// TestAuditURLsReportsEveryUnsafeSiteInBlockOrder proves AuditURLs is the
// deterministic coverage oracle its doc comment claims: exactly the sites
// in wantAuditSites, in that order, nothing more, nothing less. The
// writers' emission-time checks are judged against this same list in the
// renderhtml package's own findings test.
func TestAuditURLsReportsEveryUnsafeSiteInBlockOrder(t *testing.T) {
	got := AuditURLs(auditFixture())
	if !reflect.DeepEqual(got, wantAuditSites) {
		t.Errorf("AuditURLs mismatch.\n--- got ---\n%+v\n--- want ---\n%+v", got, wantAuditSites)
	}

	// Determinism: a second walk of the same tree returns the identical
	// slice — no map iteration order, no time-dependence.
	again := AuditURLs(auditFixture())
	if !reflect.DeepEqual(got, again) {
		t.Errorf("two AuditURLs walks of the same tree disagree:\n%+v\nvs\n%+v", got, again)
	}
}

// TestAuditURLsSilentOnSafeTrees proves the oracle's negative space: a
// fully safe tree (every href on the EM110 allowlist, every img src an
// absolute https URL, Columns without images, a Custom subtree whose only
// <a> is mailto) yields zero sites — so a writer rendering this tree must
// also produce zero URL findings, the invariant the renderhtml parity
// test asserts end to end.
func TestAuditURLsSilentOnSafeTrees(t *testing.T) {
	resolved := &Resolved{
		Blocks: []ResolvedBlock{
			ResolvedCTA{Label: "Visit", Href: "https://example.com/visit"},
			ResolvedButton{Variant: "primary", Label: "Write us", Href: "mailto:hi@example.com"},
			ResolvedHero{Src: "https://cdn.example/hero.png", Alt: "safe hero"},
			ResolvedColumns{Columns: []ResolvedColumn{{Title: "text only"}}},
			ResolvedCustom{Root: ResolvedCustomNode{
				Tag: "p",
				Children: []ResolvedCustomNode{
					{Tag: "a",
						Attrs:    []ResolvedCustomAttr{{Name: "href", Value: "mailto:list@example.com"}},
						Children: []ResolvedCustomNode{{IsText: true, Text: "subscribe"}}},
					{Tag: "img",
						Attrs: []ResolvedCustomAttr{{Name: "src", Value: "https://cdn.example/safe.png"}, {Name: "alt", Value: "safe"}}},
				},
			}},
		},
	}
	if got := AuditURLs(resolved); len(got) != 0 {
		t.Errorf("fully safe tree produced %d audit sites, want 0:\n%+v", len(got), got)
	}
}
