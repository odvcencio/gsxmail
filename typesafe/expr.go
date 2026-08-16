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
// an enclosing <Each as="name"> introduces, string/int/float literals,
// string + concatenation, comparisons on scalars, len(), and calls to a
// helper registered in helpers. templateName names the template being
// checked (EM012's message); propsName is its declared props parameter
// name. props is nil when the template declares no resolvable props type,
// in which case every props field read reports EM012. bindings maps a loop
// binding's name to its element Kind; pass nil outside an <Each> body.
// CheckExpr returns the expression's resolved Kind and the first dialect
// violation found, if any — never both.
func CheckExpr(src, templateName, propsName string, props *Props, bindings map[string]Kind, helpers map[string]any) (Kind, *Violation) {
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
// additional rule, with a fast path for the common case — a bare props
// field read — so a non-scalar field gets EM013's specific message instead
// of the generic EM010 catch-all.
func CheckInterpolation(src, templateName, propsName string, props *Props, helpers map[string]any) *Violation {
	if props != nil {
		if field, ok := BareFieldName(src, propsName); ok {
			f, exists := props.Fields[field]
			if !exists {
				return em012(templateName, props.Name, field)
			}
			if !f.Kind.IsScalar() {
				return &Violation{"EM013", fmt.Sprintf(
					"props.%s has type %s; interpolated values must be string, integer, float, or bool — format structs before rendering",
					field, f.GoType)}
			}
			return nil
		}
	}
	_, v := CheckExpr(src, templateName, propsName, props, nil, helpers)
	return v
}

type checker struct {
	templateName string
	propsName    string
	props        *Props
	bindings     map[string]Kind
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
		if k, ok := c.bindings[e.Name]; ok {
			return k, nil
		}
		return KindUnknown, em010(c.src)
	case *ast.SelectorExpr:
		if e.Sel.Name == "length" {
			return KindUnknown, em011(c.src, c.sourceSlice(e.X.Pos(), e.X.End()))
		}
		ident, ok := e.X.(*ast.Ident)
		if !ok || ident.Name != c.propsName {
			return KindUnknown, em010(c.src)
		}
		if c.props == nil {
			return KindUnknown, c.em012(e.Sel.Name)
		}
		f, ok := c.props.Fields[e.Sel.Name]
		if !ok {
			return KindUnknown, c.em012(e.Sel.Name)
		}
		return f.Kind, nil
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
