package importer

import (
	"fmt"
	"strings"
	"unicode"
)

// propsField is one synthesized field on the generated props struct
// (props.go): a scalar string, a string slice (StatTable header, or an
// Each-driven row type's own Cells field), or an Each-driven slice of
// rows.
type propsField struct {
	name   string // Go field name, e.g. "Wordmark"
	goType string // "string" or "[]string" or a synthesized row type name
	// sample is the harvested value(s) props.sample.json writes for this
	// field: a string, a []string, or (for a synthesized Each row slice)
	// a []map[string]any, one entry per synthesized row struct.
	sample any
	// source is the report's provenance line for this field: the literal
	// text (or a short structural description) the value came from.
	source string
}

// rowType is one synthesized Each-driven row struct (StatTable's own
// "Items []ItemsRow { Cells []string }" shape, matching
// examples/gallery/receipt/receipt.go's own ReceiptItem convention).
type rowType struct {
	name   string // "ItemsRow"
	fields []string
}

// propsBuilder accumulates every field Import synthesizes across one
// template, assigning collision-free Go identifiers and building
// props.go's struct plus props.sample.json's fixture in one pass.
type propsBuilder struct {
	fields   []propsField
	rowTypes []rowType
	used     map[string]bool
}

func newPropsBuilder() *propsBuilder {
	return &propsBuilder{used: map[string]bool{}}
}

// field registers a scalar (string) props field with base name want
// ("Wordmark", "Value", ...), returns the collision-free Go field name
// Import should write into the generated .gsx as props.<name>, and
// records sample/source for the report and the sample fixture.
func (b *propsBuilder) field(want, sample, source string) string {
	name := b.uniqueName(want)
	b.fields = append(b.fields, propsField{name: name, goType: "string", sample: sample, source: source})
	return name
}

// sliceField registers a []string props field (StatTable's own header
// attribute is the one shipped consumer of a bare string-slice field).
func (b *propsBuilder) sliceField(want string, sample []string, source string) string {
	name := b.uniqueName(want)
	b.fields = append(b.fields, propsField{name: name, goType: "[]string", sample: sample, source: source})
	return name
}

// eachField registers an Each-driven slice field: a props field of type
// []<RowTypeName>, where <RowTypeName> is itself a synthesized struct
// with the given cell-holding field name ("Cells []string" — the shipped
// StatRow/StatTable API's own shape; see receipt.go's ReceiptItem for the
// convention this mirrors). rows is one []string per synthesized row.
func (b *propsBuilder) eachField(want, rowFieldName string, rows [][]string, source string) (fieldName, elemName string) {
	fieldName = b.uniqueName(want)
	elemName = b.uniqueTypeName(strings.TrimSuffix(fieldName, "s") + "Row")
	b.rowTypes = append(b.rowTypes, rowType{name: elemName, fields: []string{rowFieldName}})
	samples := make([]map[string]any, len(rows))
	for i, cells := range rows {
		samples[i] = map[string]any{rowFieldName: cells}
	}
	b.fields = append(b.fields, propsField{name: fieldName, goType: "[]" + elemName, sample: samples, source: source})
	return fieldName, elemName
}

func (b *propsBuilder) uniqueName(want string) string {
	name := sanitizeIdent(want)
	if name == "" {
		name = "Field"
	}
	if !b.used[name] {
		b.used[name] = true
		return name
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", name, i)
		if !b.used[candidate] {
			b.used[candidate] = true
			return candidate
		}
	}
}

func (b *propsBuilder) uniqueTypeName(want string) string {
	return b.uniqueName(want)
}

// sanitizeIdent turns want into an exported Go identifier: letters and
// digits only, title-cased at each word break, leading digit prefixed
// with "F". A source label written in shouting caps ("SUBTOTAL", "BILLED
// TO") title-cases to "Subtotal"/"BilledTo" rather than staying shouted —
// titleWord's own rule — matching the shipped gallery's own convention
// (receipt.go's PanelRow labels read "Subtotal", not "SUBTOTAL").
func sanitizeIdent(want string) string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = nil
		}
	}
	for _, r := range want {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()

	var b strings.Builder
	for _, w := range words {
		b.WriteString(titleWord(w))
	}
	out := b.String()
	if out == "" {
		return ""
	}
	if unicode.IsDigit(rune(out[0])) {
		out = "F" + out
	}
	return out
}

// titleWord title-cases one word: an all-uppercase word ("SUBTOTAL")
// becomes "Subtotal"; anything already mixed-case ("OrderID") keeps its
// own casing beyond an upper-cased first rune, so an author's own
// intentional camelCase or acronym survives.
func titleWord(w string) string {
	r := []rune(w)
	if len(r) == 0 {
		return ""
	}
	allUpper := true
	for _, c := range r {
		if unicode.IsLower(c) {
			allUpper = false
			break
		}
	}
	out := make([]rune, len(r))
	copy(out, r)
	out[0] = unicode.ToUpper(out[0])
	if allUpper {
		for i := 1; i < len(out); i++ {
			out[i] = unicode.ToLower(out[i])
		}
	}
	return string(out)
}
