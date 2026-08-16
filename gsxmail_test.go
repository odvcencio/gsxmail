package gsxmail_test

import (
	"os"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
)

// InviteProps mirrors the type declared in testdata/emails/invite.go
// (spec section 6.2). It is declared again here, rather than imported,
// because the shared fixture lives under testdata/, which the go tool
// intentionally excludes from ordinary package discovery; Set.Render
// compares only the type's name, not its identity, so this works.
type InviteProps struct {
	ShortDate string
	LongDate  string
	DraftTime string
	LeagueURL string
	Email     string
}

// inviteFixtureProps is the exact fixture the design spec's worked example
// (section 6.3, 6.4) renders.
func inviteFixtureProps() InviteProps {
	return InviteProps{
		ShortDate: "SAT · AUG 22",
		LongDate:  "Saturday, August 22",
		DraftTime: "7:00 PM ET",
		LeagueURL: "https://gridiron.draco.quest",
		Email:     "manager@example.com",
	}
}

// gridironTheme reproduces the gridiron dark palette (spec section 6.4,
// extracted from admin.go) so the T1 goldens and the T3 parity test render
// the same pixels as the hand-written invite template.
func gridironTheme() gsxmail.Theme {
	return gsxmail.Theme{
		ColorGround: "#070A16",
		ColorCard:   "#0B1024",
		ColorPanel:  "#131B3B",
		ColorBorder: "#26305A",
		ColorAccent: "#38E8FF",
		ColorInk:    "#F7F4EA",
		ColorBody:   "#C2CAE1",
		ColorMuted:  "#8995B8",
		ColorFaint:  "#5C6690",

		FontSans: "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif",
		FontMono: "'SFMono-Regular',Consolas,Menlo,monospace",

		CardWidth: 600,

		ColorScheme: "dark",
	}
}

// loadInviteSet loads the shared testdata/emails Set with Outlook "off"
// (parity mode): InviteEmail is gridiron's production invite email, and
// its own T3 DOM-parity guard (parity_test.go) pins admin.go's hand-written
// bytes. WP5.1's hardened contract is opt-in per Set, so a template that
// must stay parity-preserving picks "off" explicitly (design spec section
// 15, WP5.1's parity guard; pixel dossier section 4.2).
func loadInviteSet(t *testing.T) *gsxmail.Set {
	t.Helper()
	set, err := gsxmail.Load(os.DirFS("testdata/emails"), gsxmail.Options{Theme: gridironTheme(), Outlook: "off"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return set
}

// T1: golden pairs. Both parts must match testdata/invite.{html,txt} byte
// for byte; those files are copied verbatim from the design spec's worked
// example (section 6.3, 6.4).
func TestInviteGoldens(t *testing.T) {
	set := loadInviteSet(t)
	parts, err := set.Render("InviteEmail", inviteFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	wantHTML, err := os.ReadFile("testdata/invite.html")
	if err != nil {
		t.Fatal(err)
	}
	wantText, err := os.ReadFile("testdata/invite.txt")
	if err != nil {
		t.Fatal(err)
	}

	if parts.HTML != string(wantHTML) {
		t.Errorf("HTML part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.HTML, wantHTML)
	}
	if parts.Text != string(wantText) {
		t.Errorf("text part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.Text, wantText)
	}
}

// T2: determinism. Rendering the same Set with the same props twice must
// produce byte-identical output (spec section 7.4).
func TestInviteDeterminism(t *testing.T) {
	set := loadInviteSet(t)
	props := inviteFixtureProps()

	first, err := set.Render("InviteEmail", props)
	if err != nil {
		t.Fatalf("Render #1: %v", err)
	}
	second, err := set.Render("InviteEmail", props)
	if err != nil {
		t.Fatalf("Render #2: %v", err)
	}

	if first.HTML != second.HTML {
		t.Error("HTML differs between two renders of the same Set and props")
	}
	if first.Text != second.Text {
		t.Error("text differs between two renders of the same Set and props")
	}
}

// T4: escaping. Recipient-influenced strings carrying <script>, &, and an
// attribute-breakout attempt must escape in the HTML part; the text part
// carries the raw string (text has no markup to escape into).
func TestEscapingHTMLContent(t *testing.T) {
	set := loadInviteSet(t)
	props := InviteProps{
		ShortDate: `<script>alert(1)</script>`,
		LongDate:  `Tom & Jerry's Draft Night`,
		DraftTime: "7:00 PM ET",
		LeagueURL: "https://gridiron.draco.quest",
		Email:     "manager@example.com",
	}

	parts, err := set.Render("InviteEmail", props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(parts.HTML, "<script>alert(1)</script>") {
		t.Error("HTML part contains an unescaped <script> tag")
	}
	if !strings.Contains(parts.HTML, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Error("HTML part does not escape the recipient-influenced <script> tag")
	}
	if !strings.Contains(parts.HTML, "Tom &amp; Jerry&#39;s Draft Night") && !strings.Contains(parts.HTML, "Tom &amp; Jerry's Draft Night") {
		t.Error("HTML part does not escape &")
	}

	if !strings.Contains(parts.Text, "<script>alert(1)</script>") {
		t.Error("text part does not pass the raw <script> string through")
	}
	if !strings.Contains(parts.Text, "Tom & Jerry's Draft Night") {
		t.Error("text part does not pass the raw & through")
	}
}

// T4 (attribute context): a quote and angle brackets in the CTA's href
// must not break out of the href="..." attribute.
func TestEscapingHrefAttribute(t *testing.T) {
	set := loadInviteSet(t)
	props := InviteProps{
		ShortDate: "SAT · AUG 22",
		LongDate:  "Saturday, August 22",
		DraftTime: "7:00 PM ET",
		LeagueURL: `https://gridiron.draco.quest/"><script>alert(1)</script>`,
		Email:     "manager@example.com",
	}

	parts, err := set.Render("InviteEmail", props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(parts.HTML, `"><script>alert(1)</script>`) {
		t.Error("HTML part lets an attacker break out of the href attribute with a raw quote")
	}
	if !strings.Contains(parts.HTML, "&quot;&gt;&lt;script&gt;") {
		t.Error("HTML part does not escape the quote and angle brackets inside the href attribute")
	}
}

// T4 (scheme allowlist): a javascript: URL is not on the CTA href
// allowlist (https, http, mailto). The CTA renders its label without a
// clickable link rather than emitting an unsafe href.
func TestEscapingUnsafeHrefScheme(t *testing.T) {
	set := loadInviteSet(t)
	props := InviteProps{
		ShortDate: "SAT · AUG 22",
		LongDate:  "Saturday, August 22",
		DraftTime: "7:00 PM ET",
		LeagueURL: "javascript:alert(document.cookie)",
		Email:     "manager@example.com",
	}

	parts, err := set.Render("InviteEmail", props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(parts.HTML, "javascript:") {
		t.Error("HTML part emits a javascript: URL; the CTA href scheme allowlist did not reject it")
	}
	if strings.Contains(parts.HTML, "<a href=") {
		t.Error("HTML part still emits a clickable <a> for the CTA despite the unsafe scheme")
	}
	if strings.Contains(parts.Text, "javascript:") {
		t.Error("text part emits a javascript: URL; the CTA href scheme allowlist did not reject it")
	}
}
