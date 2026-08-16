package importer

import (
	"strings"

	"m31labs.dev/gsxmail/lint"
)

// styleDecl is one parsed "prop: value" pair from a style attribute.
type styleDecl struct {
	prop  string // lowercased CSS property name
	value string // trimmed value, original case
}

// parseStyleDecls splits a style attribute's raw text into declarations.
// It is deliberately tolerant (a legacy source's style attribute is not
// guaranteed to be well-formed CSS): a declaration missing its ":" is
// skipped rather than failing the whole parse.
func parseStyleDecls(raw string) []styleDecl {
	var out []styleDecl
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, ":")
		if idx < 0 {
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(part[:idx]))
		value := strings.TrimSpace(part[idx+1:])
		if prop == "" || value == "" {
			continue
		}
		out = append(out, styleDecl{prop: prop, value: value})
	}
	return out
}

// matrix is the same embedded caniemail snapshot package lint's own
// EM101/EM102 checks use. Reusing it here, rather than hand-rolling a
// second unsupported-property list, is deliberate: a Custom fallback
// block's sanitized style must never trip the very check that would fail
// gsxmail.Load on the generated .gsx (proof-bar item (c): "every imported
// .gsx must LOAD cleanly").
var matrix = lint.DefaultMatrix()

// safeStyleValue reports whether every client the embedded matrix has an
// opinion on rates prop "y" or "a" (partial). A "n" (unsupported)
// property anywhere is dropped from a Custom subtree's sanitized style —
// EM101 is error-severity, and proof-bar item (c) requires the imported
// template to load with, at most, warn-severity findings. A "a"-only
// property is kept: it earns EM102 (warn), which the proof bar allows.
// A property the matrix has never heard of (most of them: color,
// font-family, font-size, padding's own longhands under some clients,
// and so on) is kept unconditionally — package lint's own checkStyle has
// nothing to say about it either.
func safeStyleValue(prop string) bool {
	for _, f := range matrix.Check(prop) {
		if f.Code == "n" {
			return false
		}
	}
	return true
}

// sanitizeStyle filters raw's declarations to the ones safeStyleValue
// allows, re-joining them in source order. It is the one place a Custom
// fallback block's literal style text passes through before generation
// (gen_gsx.go's writeCustomStyle), so every raw style attribute the
// importer ever emits is provably lint-safe by construction, not by
// review.
func sanitizeStyle(raw string) (kept string, dropped []string) {
	decls := parseStyleDecls(raw)
	var b strings.Builder
	for _, d := range decls {
		if !safeStyleValue(d.prop) {
			dropped = append(dropped, d.prop)
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(d.prop)
		b.WriteString(":")
		b.WriteString(d.value)
		b.WriteString(";")
	}
	return b.String(), dropped
}
