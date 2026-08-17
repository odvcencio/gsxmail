package gsxmail_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"m31labs.dev/gsxmail"
)

// TestDynamicPreheaderTruncates is the launch-gate M4 finding's own
// proof: EM171 only ever rejects a *static* preheader literal over 150
// runes at Load time — a dynamic {expression} preheader's own length is
// not known until a real props value resolves it at Render. Before this
// fix, an over-length dynamic preheader rendered in full, past the
// padded-to-150 contract writePreheader's own doc comment states; it now
// truncates to exactly 150 runes and reports EM200 in Parts.Diagnostics.
func TestDynamicPreheaderTruncates(t *testing.T) {
	set := loadNoticeSet(t, gsxmail.Options{})
	props := noticeFixtureProps()
	props.Preheader = strings.Repeat("x", 200)

	parts, err := set.Render("Notice", props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(parts.HTML, strings.Repeat("x", 151)) {
		t.Error("HTML part still carries more than 150 consecutive preheader runes; the render-time truncation did not apply")
	}
	if !strings.Contains(parts.HTML, strings.Repeat("x", 150)) {
		t.Error("HTML part does not carry the truncated 150-rune preheader prefix at all")
	}

	var found bool
	for _, d := range parts.Diagnostics {
		if d.Code == "EM200" && d.Severity == "warn" {
			found = true
			if !strings.Contains(d.Message, "200") || !strings.Contains(d.Message, "150") {
				t.Errorf("EM200 message %q does not name both the original and truncated lengths", d.Message)
			}
		}
	}
	if !found {
		t.Errorf("Parts.Diagnostics has no EM200 warning for the truncated preheader; got:\n%s", diagnosticsList(parts.Diagnostics))
	}
}

// TestDynamicPreheaderUnderLimitDoesNotTruncate is the companion
// negative case: a dynamic preheader within the 150-rune limit renders
// unchanged, with no EM200.
func TestDynamicPreheaderUnderLimitDoesNotTruncate(t *testing.T) {
	set := loadNoticeSet(t, gsxmail.Options{})
	props := noticeFixtureProps()
	props.Preheader = "well within the limit"

	parts, err := set.Render("Notice", props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(parts.HTML, props.Preheader) {
		t.Error("HTML part is missing the untouched preheader text")
	}
	for _, d := range parts.Diagnostics {
		if d.Code == "EM200" {
			t.Errorf("unexpected EM200 for a preheader well under the limit: %s", d.String())
		}
	}
	if utf8.RuneCountInString(props.Preheader) >= 150 {
		t.Fatal("sanity: fixture preheader must be under the 150-rune limit")
	}
}
