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
	ColorScheme string
}

// DefaultTheme is a neutral light theme: it is the OSS quick start's
// default so a fresh gsxmail project does not look like any one product's
// brand. Consumers with a brand palette (dark or light) build their own
// Theme value; nothing here is special-cased.
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
	}
}
