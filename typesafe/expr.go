package typesafe

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// Violation is one email-dialect rule an expression breaks: an EM code plus
// its exact spec-section-8 message. It carries no source position; package
// lint attaches File/Line/Col from the enclosing ir.Node, since neither
// ir.Attr nor an expression's own AST carries one gosx stamps.
type Violation struct {
	Code    string
	Message string
}

func (v *Violation) Error() string { return v.Code + ": " + v.Message }

// CheckExpr type-checks src — a raw Go expression, as recorded on an
// ir.Attr's Expr field or an ir.Node's NodeExpr Text — against the v1 email
// dialect (design spec section 6.1): props field reads, the loop bindings
// an enclosing <Each as="name"> introduces (including one level of field
// reads on a struct-element binding, "row.Cells"), string/int/float
// literals, string + concatenation, comparisons on scalars, len(), and
// calls to a helper registered in helpers. templateName names the template
// being checked (EM012's message); propsName is its declared props
// parameter name. props is nil when the template declares no resolvable
// props type, in which case every props field read reports EM012.
// bindings maps a loop binding's name to its element Binding; pass nil
// outside an <Each> body. CheckExpr returns the expression's resolved Kind
// and the first dialect violation found, if any — never both.
func CheckExpr(src, templateName, propsName string, props *Props, bindings map[string]Binding, helpers map[string]any) (Kind, *Violation) {
	fset := token.NewFileSet()
	expr, err := parser.ParseExprFrom(fset, "expr", src, 0)
	if err != nil {
		return KindUnknown, em010(src)
	}
	c := &checker{
		templateName: templateName,
		propsName:    propsName,
		props:        props,
		bindings:     bindings,
		helpers:      helpers,
		src:          src,
		fset:         fset,
	}
	return c.eval(expr)
}

// CheckInterpolation type-checks src as an attribute-hole or text-hole
// expression whose resolved value must be interpolatable text (spec
// EM013): a string, integer, float, or bool. It is CheckExpr plus that one
// additional rule, with a fast path for the common case — a bare props or
// loop-binding field read — so a non-scalar field gets EM013's specific
// message instead of the generic EM010 catch-all. bindings is the same
// loop-binding scope CheckExpr takes; pass nil outside an <Each> body.
func CheckInterpolation(src, templateName, propsName string, props *Props, bindings map[string]Binding, helpers map[string]any) *Violation {
	if root, field, ok := ParseBareSelector(src); ok && field != "" {
		if f, resolved := ResolveFieldPath(root, field, propsName, props, bindings); resolved {
			if !f.Kind.IsScalar() {
				typeName := f.GoType
				return &Violation{"EM013", fmt.Sprintf(
					"props.%s has type %s; interpolated values must be string, integer, float, or bool — format structs before rendering",
					field, typeName)}
			}
			return nil
		}
		if root == propsName {
			return em012(templateName, propsTypeName(props), field)
		}
	}
	_, v := CheckExpr(src, templateName, propsName, props, bindings, helpers)
	return v
}

func propsTypeName(props *Props) string {
	if props == nil {
		return ""
	}
	return props.Name
}

// CheckSlicePath type-checks src as a bare slice-valued path — the shape
// <Each of={...}>, <email.StatTable header={...}>, and <email.StatRow
// cells={...}> all require (design spec section 8, EM032; section 15,
// WP3): a bare props or loop-binding field read whose resolved Kind is
// KindSlice. label names the attribute for the message, exactly as it
// should read inside "<%s> requires a slice or array props path" — "Each
// of" reproduces EM032's original, pinned wording for <Each>; StatTable
// and StatRow's callers pass their own tag and attribute name so the same
// rule reads correctly for header and cells too. CheckSlicePath returns
// the slice element's own Binding (so a caller such as <Each> can extend
// its own bindings map with it) and the first violation found, if any.
func CheckSlicePath(label, src, templateName, propsName string, props *Props, bindings map[string]Binding) (elem Binding, v *Violation) {
	root, field, ok := ParseBareSelector(src)
	if !ok {
		return Binding{}, emSlicePath(label, src, "a non-props-path expression")
	}
	f, resolved := ResolveFieldPath(root, field, propsName, props, bindings)
	switch {
	case !resolved && root == propsName:
		return Binding{}, em012(templateName, propsTypeName(props), field)
	case !resolved:
		return Binding{}, emSlicePath(label, src, "a non-props-path expression")
	case f.Kind != KindSlice:
		return Binding{}, emSlicePath(label, src, f.GoType)
	default:
		return Binding{Kind: f.ElemKind, Fields: f.ElemProps}, nil
	}
}

func emSlicePath(label, src, got string) *Violation {
	return &Violation{"EM032", fmt.Sprintf(
		"<%s={%s}> requires a slice or array props path; got %s", label, src, got)}
}

type checker struct {
	templateName string
	propsName    string
	props        *Props
	bindings     map[string]Binding
	helpers      map[string]any
	src          string
	fset         *token.FileSet
}

func em010(src string) *Violation {
	return &Violation{"EM010", fmt.Sprintf(
		"expression %q is not in the email dialect; allowed: props paths, loop bindings, literals, string +, comparisons, len(), registered helpers",
		src)}
}

func em011(src, receiver string) *Violation {
	return &Violation{"EM011", fmt.Sprintf(
		"expression %q reads .length; this is not Go — use len(%s)", src, receiver)}
}

func em012(templateName, typeName, field string) *Violation {
	if typeName == "" {
		typeName = "<no props type>"
	}
	return &Violation{"EM012", fmt.Sprintf(
		"template %s reads props.%s but type %s has no field %s", templateName, field, typeName, field)}
}

func em014(name string) *Violation {
	return &Violation{"EM014", fmt.Sprintf("helper %s is not registered in Options.Helpers", name)}
}

func em015(name string, want, got int) *Violation {
	return &Violation{"EM015", fmt.Sprintf("helper %s takes %d arguments; the template passes %d", name, want, got)}
}

func (c *checker) sourceSlice(from, to token.Pos) string {
	f := c.fset.File(from)
	if f == nil {
		return ""
	}
	start, end := f.Offset(from), f.Offset(to)
	if start < 0 || end > len(c.src) || start > end {
		return ""
	}
	return c.src[start:end]
}

func (c *checker) em012(field string) *Violation {
	typeName := ""
	if c.props != nil {
		typeName = c.props.Name
	}
	return em012(c.templateName, typeName, field)
}

func (c *checker) eval(n ast.Expr) (Kind, *Violation) {
	switch e := n.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.STRING:
			return KindString, nil
		case token.INT:
			return KindInt, nil
		case token.FLOAT:
			return KindFloat, nil
		}
		return KindUnknown, em010(c.src)
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return KindBool, nil
		}
		if b, ok := c.bindings[e.Name]; ok {
			return b.Kind, nil
		}
		return KindUnknown, em010(c.src)
	case *ast.SelectorExpr:
		if e.Sel.Name == "length" {
			return KindUnknown, em011(c.src, c.sourceSlice(e.X.Pos(), e.X.End()))
		}
		ident, ok := e.X.(*ast.Ident)
		if !ok {
			return KindUnknown, em010(c.src)
		}
		if ident.Name == c.propsName {
			if c.props == nil {
				return KindUnknown, c.em012(e.Sel.Name)
			}
			f, ok := c.props.Fields[e.Sel.Name]
			if !ok {
				return KindUnknown, c.em012(e.Sel.Name)
			}
			return f.Kind, nil
		}
		if b, ok := c.bindings[ident.Name]; ok {
			if b.Fields == nil {
				return KindUnknown, em010(c.src)
			}
			f, ok := b.Fields.Fields[e.Sel.Name]
			if !ok {
				return KindUnknown, em010(c.src)
			}
			return f.Kind, nil
		}
		return KindUnknown, em010(c.src)
	case *ast.BinaryExpr:
		return c.evalBinary(e)
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			k, v := c.eval(e.X)
			if v != nil {
				return KindUnknown, v
			}
			if k != KindBool {
				return KindUnknown, em010(c.src)
			}
			return KindBool, nil
		}
		return KindUnknown, em010(c.src)
	case *ast.ParenExpr:
		return c.eval(e.X)
	case *ast.CallExpr:
		return c.evalCall(e)
	default:
		return KindUnknown, em010(c.src)
	}
}

func (c *checker) evalBinary(e *ast.BinaryExpr) (Kind, *Violation) {
	lk, v := c.eval(e.X)
	if v != nil {
		return KindUnknown, v
	}
	rk, v := c.eval(e.Y)
	if v != nil {
		return KindUnknown, v
	}
	switch e.Op {
	case token.ADD:
		if lk != KindString || rk != KindString {
			return KindUnknown, em010(c.src)
		}
		return KindString, nil
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		if !lk.IsScalar() || !rk.IsScalar() {
			return KindUnknown, em010(c.src)
		}
		return KindBool, nil
	default:
		return KindUnknown, em010(c.src)
	}
}

func (c *checker) evalCall(e *ast.CallExpr) (Kind, *Violation) {
	fn, ok := e.Fun.(*ast.Ident)
	if !ok {
		return KindUnknown, em010(c.src)
	}
	if fn.Name == "len" {
		if len(e.Args) != 1 {
			return KindUnknown, em010(c.src)
		}
		ak, v := c.eval(e.Args[0])
		if v != nil {
			return KindUnknown, v
		}
		if ak != KindSlice && ak != KindString {
			return KindUnknown, em010(c.src)
		}
		return KindInt, nil
	}

	fnVal, registered := c.helpers[fn.Name]
	if !registered {
		return KindUnknown, em014(fn.Name)
	}
	arity, retKind, ok := helperSignature(fnVal)
	if !ok {
		return KindUnknown, em010(c.src)
	}
	if len(e.Args) != arity {
		return KindUnknown, em015(fn.Name, arity, len(e.Args))
	}
	for _, a := range e.Args {
		if _, v := c.eval(a); v != nil {
			return KindUnknown, v
		}
	}
	return retKind, nil
}
