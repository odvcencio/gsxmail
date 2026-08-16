// Package welcome is the gallery's onboarding template (pixel dossier
// section 8.1): Shell, Headline, PickList, Button, Footer.
package welcome

// WelcomeProps fills WelcomeEmail.
type WelcomeProps struct {
	Product  string // "Acme" — shown in the Shell wordmark and the lede
	Name     string // "Ada" — the new user's display name
	LoginURL string // CTA target; must be https, http, or mailto
}
