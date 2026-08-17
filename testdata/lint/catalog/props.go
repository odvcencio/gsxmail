// Package lintcatalog is the email lint's own fixture set: one minimal
// .gsx function per EM-numbered rule, each shaped to trigger exactly
// that rule and (so far as the fixture allows) no other.
// gsxmail_test.go's TestLintCatalog loads this whole directory once and
// asserts that every expected (Code, Message) pair appears somewhere in
// the resulting *gsxmail.LintError.
package lintcatalog

// Props is the shared props type every fixture that needs one props-checked
// field draws from.
type Props struct {
	Name      string
	DraftTime string
	Items     []string
	Roster    Roster
}

// Roster is Props.Roster's type: a struct, so reading it bare is EM013's
// fixture (interpolating a non-scalar value).
type Roster struct {
	TeamName string
}
