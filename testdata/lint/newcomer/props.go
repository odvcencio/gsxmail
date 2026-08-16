// Package newcomer is the launch-gate B4 finding's reproduction fixture
// (see welcome.gsx): a newcomer's first template, unchanged from the
// examiner's own probe, now expected to report every attribute mistake it
// contains instead of passing check silently.
package newcomer

// WelcomeProps mirrors the probe's own declared props type.
type WelcomeProps struct {
	Name     string
	Product  string
	LoginURL string
}
