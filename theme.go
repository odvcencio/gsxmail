package gsxmail

import "m31labs.dev/gsxmail/renderhtml"

// Theme carries the palette, fonts, and metrics the HTML writer inlines
// into every element's style attribute. Themes have no effect on the text
// part.
type Theme = renderhtml.Theme

// DefaultTheme returns a neutral light theme: the OSS quick start's
// default, so a fresh gsxmail project does not carry any one product's
// brand.
func DefaultTheme() Theme {
	return renderhtml.DefaultTheme()
}
