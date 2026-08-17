// Package magiclink is the gallery's sign-in-code template: Shell,
// Headline, Panel (mono OTP row), Note (expiry), Button.
package magiclink

// MagicLinkProps fills MagicLinkEmail. Code is the one-time code as
// plain text — never rendered as an image, so it stays selectable and
// screen-reader legible.
type MagicLinkProps struct {
	Product    string // "Acme"
	Email      string // the recipient's own address, echoed back for confirmation
	Code       string // "482 913" — a short-lived sign-in code
	ExpiryNote string // "This code expires in 10 minutes."
	LoginURL   string // the full sign-in link; the code above is the fallback path
}
