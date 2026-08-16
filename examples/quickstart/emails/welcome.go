// Package emails holds the quickstart example's one template.
package emails

// WelcomeProps fills the welcome email. The caller formats every field
// before rendering; gsxmail only owns markup.
type WelcomeProps struct {
	Name     string // "Ada" — the new user's display name
	Product  string // "Acme" — the product name
	LoginURL string // CTA target; must be https, http, or mailto
}
