package doc

import "strings"

// textSafe collapses every run of control characters (including \r, \n,
// and \t) in s into a single space. A resolved field that carries an
// embedded newline could otherwise forge extra lines in the text part, or
// smuggle raw control bytes into the HTML part, that the paired part never
// shows. Ported from gridiron's internal/emailkit/text.go (a 2026 security
// review item there); resolveExpr routes every resolved value through it.
// The fast path returns s unchanged when it carries no control character,
// avoiding an allocation for the overwhelmingly common case.
func textSafe(s string) string {
	clean := true
	for _, r := range s {
		if isControlRune(r) {
			clean = false
			break
		}
	}
	if clean {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if isControlRune(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = r == ' '
		b.WriteRune(r)
	}
	return b.String()
}

// isControlRune reports whether r is a C0 control character, including
// \r, \n, and \t.
func isControlRune(r rune) bool {
	return r < 0x20 || r == 0x7f
}
