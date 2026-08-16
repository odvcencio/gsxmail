package doc

// Expr is one compiled template value: a concatenation of literal text and
// props-field reads, in source order. A plain string attribute lowers to a
// single-literal Expr; `props.LongDate + " · " + props.DraftTime` lowers to
// three parts. Resolve evaluates it against a concrete props value.
type Expr struct {
	Parts []ExprPart
}

// ExprPart is one term of an Expr. A part with Field set reads that field
// from props; otherwise Literal is emitted as-is.
type ExprPart struct {
	Literal string
	Field   string
}

// Static builds an Expr with no props reads: it always resolves to s.
func Static(s string) Expr {
	return Expr{Parts: []ExprPart{{Literal: s}}}
}
