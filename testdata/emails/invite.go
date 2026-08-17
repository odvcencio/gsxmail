// Package emails holds the gsxmail test corpus's templates: the props
// types live in plain .go files beside the .gsx template that uses
// them, resolved through go/types by package typesafe.
package emails

// InviteProps fills the invite email. All fields are already formatted
// strings; the caller owns formatting, gsxmail owns markup.
type InviteProps struct {
	ShortDate string // "SAT · AUG 22"
	LongDate  string // "Saturday, August 22"
	DraftTime string // "7:00 PM ET"
	LeagueURL string // CTA target; https enforced by lint EM110
	Email     string // recipient address; attacker-influenced
}
