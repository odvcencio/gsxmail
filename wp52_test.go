package gsxmail_test

import (
	"os"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
	"m31labs.dev/gsxmail/internal/structverify"
)

// noticeFixtureProps is the shared fixture for every WP5.2 Notice golden
// (design spec section 15, WP5.2). Preheader is left empty here: the
// preheader-on golden sets it explicitly, so the dark-mode goldens stay
// preheader-free and each shape's own diff stays isolated.
func noticeFixtureProps() NoticeProps {
	return NoticeProps{
		Wordmark:   "NOTICE CO",
		ShortCode:  "NTC",
		Tagline:    "SYSTEM ALERTS",
		Title:      "System Notice",
		HeadTitle:  "EVERYTHING'S GREEN.",
		HeadLede:   "No incidents in the last seven days. Full report attached below.",
		CTALabel:   "VIEW REPORT →",
		CTAHref:    "https://notice.example/report",
		FooterSign: "— The Systems Team",
		FooterNote: "NOTICE CO · Automated message, no reply needed.",
	}
}

// NoticeProps mirrors testdata/wp52/emails/notice.go's declared props type.
type NoticeProps struct {
	Wordmark   string
	ShortCode  string
	Tagline    string
	Title      string
	Preheader  string
	HeadTitle  string
	HeadLede   string
	CTALabel   string
	CTAHref    string
	FooterSign string
	FooterNote string
}

func loadNoticeSet(t *testing.T, opts gsxmail.Options) *gsxmail.Set {
	t.Helper()
	set, err := gsxmail.Load(os.DirFS("testdata/wp52/emails"), opts)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return set
}

// darkLockedTheme is a dark-native Theme with DarkMode "locked" (pixel
// dossier section 5.2): the theme's own fields ARE the dark presentation,
// matching the shape gridiron's own production palette already uses.
// Colors are deliberately not pure black/white and clear WCAG AA contrast
// (EM141/EM142), proving a "Terminal"-style dark theme — the pixel
// dossier's own WP5.3 gallery theme (section 8.2), not shipped here —
// passes both checks without needing to import that later work package.
func darkLockedTheme() gsxmail.Theme {
	return gsxmail.Theme{
		ColorGround: "#0C100D",
		ColorCard:   "#101611",
		ColorPanel:  "#16201A",
		ColorBorder: "#23402F",
		ColorAccent: "#33E68C",
		ColorInk:    "#E8F5EC",
		ColorBody:   "#C7D9CE",
		ColorMuted:  "#7FA28D",
		ColorFaint:  "#4F6D5C",

		FontSans: "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif",
		FontMono: "'SFMono-Regular',Consolas,Menlo,monospace",

		CardWidth: 600,

		ColorScheme: "dark",
		DarkMode:    "locked",
	}
}

// adaptiveDarkTheme is a light-native Theme (Paper's own tokens) with
// DarkMode "adaptive" and a Dark palette. The Dark palette's ColorCard,
// ColorInk, and ColorMuted values are the pixel dossier's own section
// 5.2 adaptive worked example, verbatim, so the golden this theme
// produces matches the dossier's cited numbers directly.
func adaptiveDarkTheme() gsxmail.Theme {
	theme := gsxmail.DefaultTheme()
	theme.DarkMode = "adaptive"
	theme.Dark = &gsxmail.DarkPalette{
		ColorGround: "#0A0A0D",
		ColorCard:   "#101014",
		ColorPanel:  "#17171C",
		ColorBorder: "#2A2A33",
		ColorAccent: "#6E8CFF",
		ColorInk:    "#F2F2F5",
		ColorBody:   "#D6D6DE",
		ColorMuted:  "#9C9CA8",
		ColorFaint:  "#6B6B76",
	}
	return theme
}

// TestNoticePreheaderGolden is the "preheader-on" golden variant (design
// spec section 15, WP5.2): the default Paper theme, hardened mode, a
// preheader set on the Shell. Both the HTML byte-exact golden and the
// full structural pass (including the new EM173 preheader-stack check)
// must hold.
func TestNoticePreheaderGolden(t *testing.T) {
	set := loadNoticeSet(t, gsxmail.Options{})
	props := noticeFixtureProps()
	props.Preheader = "Your weekly system digest is ready to review"

	parts, err := set.Render("Notice", props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	wantHTML, err := os.ReadFile("testdata/wp52/notice_preheader.html")
	if err != nil {
		t.Fatal(err)
	}
	if parts.HTML != string(wantHTML) {
		t.Errorf("HTML part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.HTML, wantHTML)
	}

	findings, err := structverify.VerifyContract(parts.HTML)
	if err != nil {
		t.Fatalf("VerifyContract: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("Notice (preheader-on) fails the structural pass:\n%s", joinFindings(findings))
	}
}

// TestNoticePreheaderPadIsRawUTF8 is polish item 11's own proof: the
// preheader pad tail is raw UTF-8 (NBSP, ZWNJ), not the "&nbsp;&zwnj;"
// entity spelling — 5 bytes per pair instead of 12, the same two decoded
// characters, cheaper on the wire. The document already declares UTF-8.
func TestNoticePreheaderPadIsRawUTF8(t *testing.T) {
	set := loadNoticeSet(t, gsxmail.Options{})
	props := noticeFixtureProps()
	props.Preheader = "Your weekly system digest is ready to review"

	parts, err := set.Render("Notice", props)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(parts.HTML, "&nbsp;") || strings.Contains(parts.HTML, "&zwnj;") {
		t.Error("HTML part still carries the entity-spelled pad tail; want raw UTF-8 NBSP/ZWNJ")
	}
	if !strings.Contains(parts.HTML, " ‌") {
		t.Error("HTML part is missing the raw NBSP+ZWNJ pad pair")
	}
}

// TestNoticeDarkLockedGolden is the "hardened+dark" (locked strategy)
// golden variant.
func TestNoticeDarkLockedGolden(t *testing.T) {
	set := loadNoticeSet(t, gsxmail.Options{Theme: darkLockedTheme()})
	parts, err := set.Render("Notice", noticeFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	wantHTML, err := os.ReadFile("testdata/wp52/notice_dark_locked.html")
	if err != nil {
		t.Fatal(err)
	}
	if parts.HTML != string(wantHTML) {
		t.Errorf("HTML part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.HTML, wantHTML)
	}

	findings, err := structverify.VerifyContract(parts.HTML)
	if err != nil {
		t.Fatalf("VerifyContract: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("Notice (dark locked) fails the structural pass:\n%s", joinFindings(findings))
	}
}

// TestNoticeDarkAdaptiveGolden is the "hardened+dark" (adaptive strategy)
// golden variant: the @media(prefers-color-scheme:dark) layer, the
// [data-ogsc]/[data-ogsb] hooks, and the gsx-ink/gsx-muted class markers
// on the Shell wordmark/tagline and Headline title.
func TestNoticeDarkAdaptiveGolden(t *testing.T) {
	set := loadNoticeSet(t, gsxmail.Options{Theme: adaptiveDarkTheme()})
	parts, err := set.Render("Notice", noticeFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	wantHTML, err := os.ReadFile("testdata/wp52/notice_dark_adaptive.html")
	if err != nil {
		t.Fatal(err)
	}
	if parts.HTML != string(wantHTML) {
		t.Errorf("HTML part does not match the golden.\n--- got ---\n%s\n--- want ---\n%s", parts.HTML, wantHTML)
	}

	findings, err := structverify.VerifyContract(parts.HTML)
	if err != nil {
		t.Fatalf("VerifyContract: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("Notice (dark adaptive) fails the structural pass:\n%s", joinFindings(findings))
	}
}

// TestNoticeOutlookShellOverride proves the WP5.2 per-Shell outlook
// surface overrides the Set-level Options.Outlook default/fallback in
// both directions (design spec section 15, WP5.2's Shell options surface).
func TestNoticeOutlookShellOverride(t *testing.T) {
	// Set default is hardened (""); NoticeOutlookOff's own Shell attribute
	// selects parity mode for that one template.
	hardSet := loadNoticeSet(t, gsxmail.Options{})
	parityParts, err := hardSet.Render("NoticeOutlookOff", noticeFixtureProps())
	if err != nil {
		t.Fatalf("Render NoticeOutlookOff: %v", err)
	}
	if strings.Contains(parityParts.HTML, "<!--[if mso | IE]") {
		t.Error("NoticeOutlookOff still emits the Outlook ghost table; its outlook=\"off\" Shell attribute did not override the Set's hardened default")
	}
	// The sibling Notice template in the same Set, with no Shell override,
	// still renders hardened — proving the override is per-template, not
	// Set-wide.
	hardParts, err := hardSet.Render("Notice", noticeFixtureProps())
	if err != nil {
		t.Fatalf("Render Notice: %v", err)
	}
	if !strings.Contains(hardParts.HTML, "<!--[if mso | IE]") {
		t.Error("Notice (no Shell override) lost the Set's hardened default")
	}

	// Set default is parity ("off"); NoticeOutlookGhostTables' own Shell
	// attribute selects the hardened contract for that one template.
	paritySet := loadNoticeSet(t, gsxmail.Options{Outlook: "off"})
	hardOverrideParts, err := paritySet.Render("NoticeOutlookGhostTables", noticeFixtureProps())
	if err != nil {
		t.Fatalf("Render NoticeOutlookGhostTables: %v", err)
	}
	if !strings.Contains(hardOverrideParts.HTML, "<!--[if mso | IE]") {
		t.Error("NoticeOutlookGhostTables did not override the Set's parity default with its own outlook=\"ghost-tables\" Shell attribute")
	}
	parityDefaultParts, err := paritySet.Render("Notice", noticeFixtureProps())
	if err != nil {
		t.Fatalf("Render Notice: %v", err)
	}
	if strings.Contains(parityDefaultParts.HTML, "<!--[if mso | IE]") {
		t.Error("Notice (no Shell override) lost the Set's parity default")
	}
}

// TestNoticeCheckReportsEM170 proves Load's Set.Check() surfaces EM170 for
// NoticeOutlookOff and NoticeOutlookGhostTables (neither sets a preheader
// in this fixture), while Notice itself — which the preheader-on golden
// exercises with a non-empty Preheader — is not required to.
func TestNoticeCheckReportsEM170(t *testing.T) {
	set := loadNoticeSet(t, gsxmail.Options{})
	var found bool
	for _, d := range set.Check() {
		if d.Code == "EM170" {
			found = true
		}
	}
	if !found {
		t.Error("Set.Check() did not report EM170 for a Shell with no preheader attribute")
	}
}
