// Package lint runs the email dialect's check-time rule catalog, EM001
// through EM112 (design spec section 8), over a compiled gosx program, and
// answers caniemail client-support questions from an embedded, dated
// snapshot (EM101/EM102). gsxmail.Load runs it before lowering a single
// template; gsxmail.Set.Check exposes its findings without rendering.
package lint

import "fmt"

// Diagnostic is one check-time finding. gsxmail.Diagnostic is a type alias
// for this type, so package gsxmail never has to convert between the two.
type Diagnostic struct {
	File     string
	Line     int
	Col      int
	Code     string
	Severity string
	Message  string
}

// String formats d as "file:line:col: CODE: message" (spec section 8).
func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d:%d: %s: %s", d.File, d.Line, d.Col, d.Code, d.Message)
}
