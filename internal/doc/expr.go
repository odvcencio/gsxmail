package doc

// Expr is one compiled template value: an expression tree in the email
// dialect (design spec section 6.1). The same tree shape serves every
// expression-typed hole — text interpolation (Signal.Text, PanelRow.Value,
// ...) and the bool context <If cond> and StatRow.Mark need — because both
// contexts share one grammar: literals, a props or loop-binding field
// read, string concatenation, negation, scalar comparison, len(), and a
// call to a helper registered in Options.Helpers. Resolve's two entry
// points, ResolveText and ResolveBool, pick the evaluation each hole
// needs; ExprKind says which shapes are legal at each entry point.
type Expr struct {
	Kind ExprKind

	// Literal carries ExprLiteralString's text, and ExprLiteralNumber's
	// decimal digits kept as source text (only ExprCompare ever parses a
	// number out of it, and only when the comparison needs one — nothing
	// else in the dialect does arithmetic on a number literal).
	Literal string
	// Bool carries ExprLiteralBool's value.
	Bool bool

	// Binding names the loop variable an ExprField reads from; "" reads
	// from the template's props instead. Field is the field name to read.
	// Field == "" together with Binding != "" reads the loop variable's
	// whole bound value, for <Each> over a slice of scalars rather than
	// structs (props field reads are always named, so Field == "" never
	// applies when Binding == "").
	Binding string
	Field   string

	// Parts holds ExprConcat's operands, evaluated to text and joined in
	// source order ("a" + props.B + "c" lowers to three parts).
	Parts []Expr

	// X is ExprNot's and ExprLen's single operand.
	X *Expr

	// Op, Left, and Right are ExprCompare's operator ("==", "!=", "<",
	// "<=", ">", ">=") and its two operands.
	Op          string
	Left, Right *Expr

	// Call is ExprCall's helper name; Args are its argument expressions.
	Call string
	Args []Expr
}

// ExprKind discriminates Expr's shape.
type ExprKind int

const (
	// ExprLiteralString is a bare string literal.
	ExprLiteralString ExprKind = iota
	// ExprLiteralNumber is a bare int or float literal.
	ExprLiteralNumber
	// ExprLiteralBool is the bare identifier true or false.
	ExprLiteralBool
	// ExprField reads Field from props (Binding == "") or from the named
	// loop binding.
	ExprField
	// ExprConcat joins Parts, each evaluated to text, in order.
	ExprConcat
	// ExprNot negates X, a bool-context expression.
	ExprNot
	// ExprCompare compares Left and Right with Op.
	ExprCompare
	// ExprLen measures X's length (a string or slice field).
	ExprLen
	// ExprCall invokes the helper named Call with Args.
	ExprCall
)

// Static builds an Expr that always resolves to s, with no props or
// binding reads.
func Static(s string) Expr {
	return Expr{Kind: ExprLiteralString, Literal: s}
}

// FieldPath is a bare field reference: props.Field, or (inside an <Each>
// body) binding.Field. It is the only shape the email dialect accepts for
// a slice-valued hole — <Each of=...>, <email.StatTable header=...>, and
// <email.StatRow cells=...> — because the dialect has no way to compute a
// slice from a concatenation, a comparison, or a helper call.
type FieldPath struct {
	// Binding names the loop variable to read Field from; "" reads from
	// props.
	Binding string
	Field   string
}

// IsZero reports whether p names no field at all — StatTable.Header's "no
// header row" sentinel, and the zero value a missing optional attribute
// lowers to.
func (p FieldPath) IsZero() bool { return p.Binding == "" && p.Field == "" }
