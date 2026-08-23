package renderhtml

import (
	"m31labs.dev/gsxmail/internal/doc"
)

// hasSafeHrefScheme reports whether raw passes doc.SafeURL — the one
// centralized EM110 scheme allowlist (https, http, or mailto) both
// writers share. A CTA whose Href fails this check renders its label
// alone, un-clickable, in the same button face — the writer enforces this
// fail-closed default at render time, so an unsafe href never reaches the
// output even from a caller that skips the email lint. The check used to
// live here as a full url.Parse + switch; it moved to internal/doc so the
// text writer and this one cannot drift apart.
func hasSafeHrefScheme(raw string) bool {
	return doc.SafeURL(raw)
}
