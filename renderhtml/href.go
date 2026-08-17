package renderhtml

import (
	"net/url"
	"strings"
)

// hasSafeHrefScheme reports whether raw parses as a URL with a scheme on
// the allowlist EM110 states: https, http, or mailto. A CTA whose Href
// fails this check renders its label alone, un-clickable, in the same
// button face — the writer enforces this fail-closed default at render
// time, so an unsafe href never reaches the output even from a caller
// that skips the email lint. Ported from
// gridiron's internal/emailkit hasSafeURLScheme (a 2026 security review
// item there), widened from http/https to also allow mailto per EM110.
func hasSafeHrefScheme(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto":
		return true
	default:
		return false
	}
}
