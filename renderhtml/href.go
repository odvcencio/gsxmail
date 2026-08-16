package renderhtml

import (
	"net/url"
	"strings"
)

// hasSafeHrefScheme reports whether raw parses as a URL with a scheme on
// the allowlist the design spec's EM110 rule states: https, http, or
// mailto (section 8). A CTA whose Href fails this check renders its label
// alone, un-clickable, in the same button face — WP1 has no lint stage to
// reject the template at Load time (that is WP2's EM catalog), so the
// writer enforces the same fail-closed default at render time. Ported from
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
