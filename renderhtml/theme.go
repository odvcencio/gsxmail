package renderhtml

// Theme carries the palette, fonts, and metrics the HTML writer inlines
// into every element's style attribute. Themes never affect the text
// writer: plain text has no color or font. Theme lives here (rather than
// in the root gsxmail package, where the repo layout in the design spec
// places the public Theme name) because the writer needs the type without
// importing the root package, which itself imports this one; gsxmail.go
// re-exports Theme and DefaultTheme as a type alias and a wrapper func.
type Theme struct {
	ColorGround string // page background
	ColorCard   string // the card's own background
	ColorPanel  string // Panel row background
	ColorBorder string
	ColorAccent string // signal line, CTA face, list numerals
	ColorInk    string // headline text
	ColorBody   string // body copy
	ColorMuted  string // mono labels
	ColorFaint  string // footer fine print

	FontSans string
	FontMono string

	// CardWidth is the card's pixel width (both the width attribute and
	// the width/max-width style declarations).
	CardWidth int

	// ColorScheme, when non-empty ("dark" or "light"), emits the
	// color-scheme and supported-color-schemes meta tags declaring the
	// theme's native scheme (spec section 6.4). Empty omits both tags.
	// DarkStrategy's "locked" and "adaptive" strategies read it (and, for
	// "adaptive", override it outright) rather than duplicating a second
	// scheme field — see DarkStrategy and DarkMode's own doc comment.
	ColorScheme string

	// DarkMode selects the WP5.2 dark-mode strategy (design spec section
	// 15, WP5.2; pixel dossier section 5.2): "" and "none" (the default)
	// change nothing — Write's hardened head keeps emitting exactly the
	// ColorScheme-driven metas WP5.1 shipped, and no style layer. "locked"
	// declares this Theme itself dark-native: the hardened head adds a
	// `:root{color-scheme:...}` rule (ColorScheme if set, else "dark") to
	// the <style> block. "adaptive" swaps to Dark's tokens under
	// `@media (prefers-color-scheme:dark)` (plus best-effort
	// `[data-ogsc]`/`[data-ogsb]` Outlook-app hooks) and overrides the meta
	// pair to "light dark" regardless of ColorScheme; it requires Dark to
	// be set (EM140 rejects Load otherwise). Call DarkStrategy to read this
	// field normalized ("" treated as "none").
	//
	// No strategy here ever claims control of Gmail's forced dark
	// transform (pixel dossier section 5.1: "no strategy controls Gmail's
	// app transform. Mitigations only.") — see the README's dark-mode
	// section for the honest per-client reach each strategy has.
	DarkMode string

	// Dark carries the dark-presentation color tokens the "adaptive"
	// strategy swaps to under prefers-color-scheme. It is nil for "none"
	// and "locked" (both use Theme's own fields, or nothing). EM140 fails
	// Load closed when DarkMode is "adaptive" and Dark is nil.
	Dark *DarkPalette
}

// DarkPalette mirrors Theme's own color tokens (minus fonts, width, and
// the meta-only ColorScheme field, none of which have a second "dark"
// value) for the "adaptive" dark-mode strategy's swapped-in presentation
// (pixel dossier section 5.2).
type DarkPalette struct {
	ColorGround string
	ColorCard   string
	ColorPanel  string
	ColorBorder string
	ColorAccent string
	ColorInk    string
	ColorBody   string
	ColorMuted  string
	ColorFaint  string
}

// DarkStrategy reports t's dark-mode strategy, normalizing DarkMode's zero
// value "" to "none" so every caller — the writer, the EM140-144 theme
// checks — compares against one three-way enum instead of repeating the
// "" / "none" equivalence.
func (t Theme) DarkStrategy() string {
	if t.DarkMode == "" {
		return "none"
	}
	return t.DarkMode
}

// DefaultTheme is a neutral light theme: it is the OSS quick start's
// default so a fresh gsxmail project does not look like any one product's
// brand. Consumers with a brand palette (dark or light) build their own
// Theme value; nothing here is special-cased. Its dark-mode strategy is
// "none" (pixel dossier section 5.2's own default): a fresh project
// declares itself light-native and adds nothing further until its author
// opts into "locked" or "adaptive".
func DefaultTheme() Theme {
	return Theme{
		ColorGround: "#F4F4F6",
		ColorCard:   "#FFFFFF",
		ColorPanel:  "#F7F7F9",
		ColorBorder: "#E2E2E8",
		ColorAccent: "#3452FF",
		ColorInk:    "#16161D",
		ColorBody:   "#3C3C46",
		ColorMuted:  "#71717F",
		ColorFaint:  "#9C9CA8",

		FontSans: "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif",
		FontMono: "'SFMono-Regular',Consolas,Menlo,monospace",

		CardWidth: 600,

		ColorScheme: "",
		DarkMode:    "none",
	}
}

// TerminalTheme is a dark, mono-forward gallery theme (pixel dossier
// section 8.2, WP5.3): green-on-near-black, deliberately not the private
// gridiron aqua/navy palette (dossier section 8.2's own note; section 12,
// open question 3's default). Its DarkMode is "locked" — the theme is
// itself dark-native, the dossier's own assignment for Terminal — so
// Theme.Dark stays nil; there is no separate presentation to swap to. The
// same token values already proved EM140-144 in WP5.2's own private
// darkLockedTheme fixture (wp52_test.go), carried over here unchanged now
// that they ship as a named theme.
func TerminalTheme() Theme {
	return Theme{
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

// LedgerTheme is a warm, print-like light gallery theme (pixel dossier
// section 8.2, WP5.3). Its DarkMode is "adaptive" with a genuine Dark
// palette: unlike Terminal (already dark-native, so it needs no separate
// swapped-in presentation), Ledger is light-native, so demonstrating a
// real dark experience for it means carrying an actual companion dark
// palette. Together, the two named themes show off gsxmail's two
// non-trivial dark-mode strategies with real, shipped themes: Terminal
// is "locked" dark by nature; Ledger swaps to its own Dark palette under
// "adaptive". ColorScheme stays "" ("adaptive" computes its own "light
// dark" meta pair; EM144 rejects a stale explicit value here).
func LedgerTheme() Theme {
	return Theme{
		ColorGround: "#FBF7EF",
		ColorCard:   "#FFFFFF",
		ColorPanel:  "#F5EFE1",
		ColorBorder: "#E7DECB",
		ColorAccent: "#B4451F",
		ColorInk:    "#26201A",
		ColorBody:   "#4A4038",
		ColorMuted:  "#8A7E6C",
		ColorFaint:  "#A89A87",

		FontSans: "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif",
		FontMono: "'SFMono-Regular',Consolas,Menlo,monospace",

		CardWidth: 600,

		ColorScheme: "",
		DarkMode:    "adaptive",
		Dark: &DarkPalette{
			ColorGround: "#17130E",
			ColorCard:   "#1E1913",
			ColorPanel:  "#241E17",
			ColorBorder: "#3A2F23",
			ColorAccent: "#E08A52",
			ColorInk:    "#F3E9DA",
			ColorBody:   "#D8CAB8",
			ColorMuted:  "#A08F79",
			ColorFaint:  "#6E6153",
		},
	}
}
