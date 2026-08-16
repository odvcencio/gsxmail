package lint_test

import (
	"testing"

	"m31labs.dev/gsxmail/lint"
	"m31labs.dev/gsxmail/renderhtml"
)

// findDiag reports whether diags contains a finding with the exact
// (code, message) pair.
func findDiag(diags []lint.Diagnostic, code, message string) bool {
	for _, d := range diags {
		if d.Code == code && d.Message == message {
			return true
		}
	}
	return false
}

// TestCheckThemeCatalog is T5-style for the WP5.2 Theme-level rules
// (design spec section 15, WP5.2; pixel dossier section 5): each case
// builds one Theme shaped to trigger exactly one EM code and pins its
// exact message.
func TestCheckThemeCatalog(t *testing.T) {
	cases := []struct {
		name    string
		theme   renderhtml.Theme
		code    string
		message string
	}{
		{
			name: "EM140",
			theme: renderhtml.Theme{
				DarkMode: "adaptive", // Dark left nil on purpose
			},
			code:    "EM140",
			message: `theme sets DarkMode "adaptive" but Theme.Dark is nil; adaptive mode requires a dark palette`,
		},
		{
			name: "EM141",
			theme: renderhtml.Theme{
				DarkMode:  "locked",
				ColorCard: "#101010",
				ColorInk:  "#151515", // near-black on near-black: fails 4.5:1
				ColorBody: "#F5F5F5",
			},
			code:    "EM141",
			message: `theme contrast ColorInk on ColorCard is 1.0:1; body text requires 4.5:1 (WCAG AA)`,
		},
		{
			name: "EM142",
			theme: renderhtml.Theme{
				DarkMode:  "locked",
				ColorCard: "#FFFFFF", // pure white in a locked (dark-native) theme
				ColorInk:  "#101010",
				ColorBody: "#202020",
			},
			code:    "EM142",
			message: `theme color ColorCard is pure black or pure white; forced dark-mode transforms map extremes hard — use midtones`,
		},
		{
			name: "EM144-locked",
			theme: renderhtml.Theme{
				DarkMode:    "locked",
				ColorScheme: "light", // conflicts: locked implies dark
			},
			code:    "EM144",
			message: `meta color-scheme "light" conflicts with theme DarkMode "locked"`,
		},
		{
			name: "EM144-adaptive",
			theme: renderhtml.Theme{
				DarkMode:    "adaptive",
				ColorScheme: "dark", // adaptive computes its own "light dark"
				Dark:        &renderhtml.DarkPalette{ColorCard: "#101014", ColorInk: "#F2F2F5", ColorBody: "#D6D6DE"},
			},
			code:    "EM144",
			message: `meta color-scheme "dark" conflicts with theme DarkMode "adaptive"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags := lint.CheckTheme(tc.theme)
			if !findDiag(diags, tc.code, tc.message) {
				t.Errorf("no diagnostic %s: %q found; got: %+v", tc.code, tc.message, diags)
			}
		})
	}
}

// TestCheckThemeDefaultThemeIsClean proves DefaultTheme (Paper, DarkMode
// "none") trips none of EM140-EM144 — the dossier's own WP5.2 acceptance
// line ("Terminal and Paper pass EM141/EM142").
func TestCheckThemeDefaultThemeIsClean(t *testing.T) {
	diags := lint.CheckTheme(renderhtml.DefaultTheme())
	if len(diags) != 0 {
		t.Errorf("DefaultTheme() tripped %d theme diagnostic(s), want none: %+v", len(diags), diags)
	}
}

// TestCheckThemeLockedDarkThemePasses proves a well-formed "locked" dark
// theme (midtone colors, WCAG AA contrast, a consistent ColorScheme)
// clears EM140-EM144 — the pixel dossier's "Terminal"-style dark theme
// (section 8.2, not shipped in WP5.2; see wp52_test.go's darkLockedTheme).
func TestCheckThemeLockedDarkThemePasses(t *testing.T) {
	theme := renderhtml.Theme{
		ColorGround: "#0C100D", ColorCard: "#101611", ColorPanel: "#16201A",
		ColorBorder: "#23402F", ColorAccent: "#33E68C", ColorInk: "#E8F5EC",
		ColorBody: "#C7D9CE", ColorMuted: "#7FA28D", ColorFaint: "#4F6D5C",
		ColorScheme: "dark", DarkMode: "locked",
	}
	diags := lint.CheckTheme(theme)
	if len(diags) != 0 {
		t.Errorf("a well-formed locked dark theme tripped %d diagnostic(s), want none: %+v", len(diags), diags)
	}
}

// TestCheckThemeAdaptiveDarkPalettePasses mirrors the above for an
// "adaptive" theme's Dark palette (wp52_test.go's adaptiveDarkTheme,
// verbatim the pixel dossier's own section 5.2 example numbers).
func TestCheckThemeAdaptiveDarkPalettePasses(t *testing.T) {
	theme := renderhtml.DefaultTheme()
	theme.DarkMode = "adaptive"
	theme.Dark = &renderhtml.DarkPalette{
		ColorGround: "#0A0A0D", ColorCard: "#101014", ColorPanel: "#17171C",
		ColorBorder: "#2A2A33", ColorAccent: "#6E8CFF", ColorInk: "#F2F2F5",
		ColorBody: "#D6D6DE", ColorMuted: "#9C9CA8", ColorFaint: "#6B6B76",
	}
	diags := lint.CheckTheme(theme)
	if len(diags) != 0 {
		t.Errorf("a well-formed adaptive dark palette tripped %d diagnostic(s), want none: %+v", len(diags), diags)
	}
}

// TestCheckThemeNamedGalleryThemesPass is the WP5.3 acceptance line (pixel
// dossier section 8.2: "All three palettes must pass EM141 contrast and
// EM142 midtone lint"; the task's own "both pass EM140-144"): the two new
// named themes, called through their real renderhtml constructors rather
// than a copy of their token values, must trip none of EM140-EM144.
func TestCheckThemeNamedGalleryThemesPass(t *testing.T) {
	for _, tc := range []struct {
		name  string
		theme renderhtml.Theme
	}{
		{"Terminal", renderhtml.TerminalTheme()},
		{"Ledger", renderhtml.LedgerTheme()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := lint.CheckTheme(tc.theme)
			if len(diags) != 0 {
				t.Errorf("%sTheme() tripped %d theme diagnostic(s), want none: %+v", tc.name, len(diags), diags)
			}
		})
	}
}
