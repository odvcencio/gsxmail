// Package alert is the gallery's notification template (pixel dossier
// section 8.1): Shell, Signal, Badge, Note, Button. Its fixture highlight
// is the "locked" dark Terminal theme, and severity carried as text and
// structure, never color alone (Badge's own border-plus-label shape).
package alert

// AlertProps fills AlertEmail. Badge's tone attribute is a static,
// compile-time literal (doc.Badge's own doc comment), so AlertEmail
// selects between two fixed-tone Badges with <If cond={props.IsCritical}>
// rather than passing tone through props directly.
type AlertProps struct {
	Product     string
	Severity    string // "CRITICAL" or "WARNING" — the Badge's own label
	IsCritical  bool
	Message     string // the alert's own body text (email.Note)
	ActionLabel string
	ActionURL   string
}
