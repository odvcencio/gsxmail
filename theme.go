package gsxmail

import "m31labs.dev/gsxmail/renderhtml"

// Theme carries the palette, fonts, and metrics the HTML writer inlines
// into every element's style attribute. Themes have no effect on the text
// part.
type Theme = renderhtml.Theme

// DarkPalette carries the dark-presentation color tokens a Theme's
// DarkMode "adaptive" strategy swaps to under prefers-color-scheme (design
// spec section 15, WP5.2; pixel dossier section 5.2).
type DarkPalette = renderhtml.DarkPalette

// DefaultTheme returns a neutral light theme: the OSS quick start's
// default, so a fresh gsxmail project does not carry any one product's
// brand. Its dark-mode strategy is "none".
func DefaultTheme() Theme {
	return renderhtml.DefaultTheme()
}

// TerminalTheme returns a dark, mono-forward named theme (pixel dossier
// section 8.2, WP5.3): green-on-near-black, DarkMode "locked". It is
// deliberately not the private gridiron aqua/navy palette, which stays
// unshipped. See the README's "Named themes" section for the full
// palette and its EM140-144 proof.
func TerminalTheme() Theme {
	return renderhtml.TerminalTheme()
}

// LedgerTheme returns a warm, print-like light named theme (pixel dossier
// section 8.2, WP5.3), DarkMode "adaptive" with its own companion Dark
// palette. See the README's "Named themes" section for the full palette.
func LedgerTheme() Theme {
	return renderhtml.LedgerTheme()
}
