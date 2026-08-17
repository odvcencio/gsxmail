// Package newcomer is a fixture modeling a newcomer's first template
// (see welcome.gsx): every attribute mistake it contains is expected to
// report, instead of passing check silently.
package newcomer

// WelcomeProps mirrors the fixture's own declared props type.
type WelcomeProps struct {
	Name     string
	Product  string
	LoginURL string
}
