// Package typesafe resolves a gsxmail template's declared props struct with
// go/types and type-checks the email dialect's expression grammar (design
// spec section 6.1, section 15 WP2) against the resolved fields. It replaces
// render-time reflection for everything it can decide at Load time: which
// props fields exist, what type each one has, and whether an expression is
// in the email dialect at all. It never touches props values — only types
// and expression source text — so it has no render-time role; doc.Resolve's
// reflection-based evaluator keeps doing that job, unchanged, as the
// second, render-time layer of the same fail-closed guarantee.
package typesafe

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path"
	"strings"
)

// Kind is the resolved dialect type of a struct field or an expression, as
// far as the email dialect's rules care: which types may be interpolated
// (EM013) and which may back an <Each> loop (EM032).
type Kind int

const (
	KindUnknown Kind = iota
	KindString
	KindInt
	KindFloat
	KindBool
	KindSlice
	KindOther
)

// String names k for diagnostic messages.
func (k Kind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindBool:
		return "bool"
	case KindSlice:
		return "slice"
	default:
		return "unknown"
	}
}

// IsScalar reports whether k is one of the four types the email dialect may
// interpolate into text (spec EM013): string, integer, float, or bool.
func (k Kind) IsScalar() bool {
	switch k {
	case KindString, KindInt, KindFloat, KindBool:
		return true
	}
	return false
}

// Field is one resolved struct field.
type Field struct {
	Kind Kind
	// GoType is the field's declared Go type, printed relative to its own
	// package (so a same-package struct field prints as "Roster", not
	// "emails.Roster"), for EM013's and EM032's messages.
	GoType string
	// ElemKind and ElemType describe a KindSlice field's element type; they
	// are the zero value otherwise.
	ElemKind Kind
	ElemType string
	// ElemProps resolves ElemType's own fields when a KindSlice field's
	// element is itself a struct (nil otherwise, including for a scalar
	// element type). <Each as="row"> over such a field binds "row" to a
	// value whose own fields (row.Cells, row.IsKeystone, ...) this drives —
	// the email dialect's answer to "nested prop reads" for loop bodies
	// (design spec section 6.1), scoped to one level of nesting.
	ElemProps *Props
}

// Binding is what an <Each as="name"> loop variable resolves to for the
// rest of its body: name's own Kind (KindString for []string, KindOther
// or a scalar Kind for []T, ...), and, when the slice's element is a
// struct, that struct's own field map so the checker can validate
// name.Field reads the same way it validates props.Field reads. Fields is
// nil for a scalar-element loop (name has no fields to read).
type Binding struct {
	Kind   Kind
	Fields *Props
}

// Props is a props struct go/types resolved from the *.go files beside a
// template's .gsx source.
type Props struct {
	// Name is the struct's declared type name (EM012/EM013's "type %s").
	Name   string
	Fields map[string]Field
}

// Resolver resolves named props struct types from the *.go files in a
// template directory, parsing and type-checking each directory's package at
// most once. A Resolver is not safe for concurrent use; Load builds one per
// call and discards it afterward.
type Resolver struct {
	fsys  fs.FS
	fset  *token.FileSet
	cache map[string]*resolvedPackage
}

type resolvedPackage struct {
	pkg *types.Package
	err error
}

// NewResolver returns a Resolver that reads Go source from fsys.
func NewResolver(fsys fs.FS) *Resolver {
	return &Resolver{fsys: fsys, fset: token.NewFileSet(), cache: make(map[string]*resolvedPackage)}
}

// Resolve returns the struct type named typeName declared among the *.go
// files in dir (a template's own directory: gsxmail templates are one
// fs.FS-loadable package per directory, spec section 6.1). typeName may
// carry a leading "*" (a pointer props parameter); it is stripped before
// lookup. An empty typeName, or a typeName that is not a plain identifier
// (a map type, a generic instantiation, and so on), returns (nil, nil): the
// caller then skips every Load-time field/type check for that template and
// relies on doc.Resolve's render-time reflection path alone, exactly as
// WP1 did for every template.
func (r *Resolver) Resolve(dir, typeName string) (*Props, error) {
	typeName = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(typeName), "*"))
	if typeName == "" || !isPlainIdent(typeName) {
		return nil, nil
	}

	rp := r.packageFor(dir)
	if rp.err != nil {
		return nil, rp.err
	}
	obj := rp.pkg.Scope().Lookup(typeName)
	if obj == nil {
		return nil, fmt.Errorf("gsxmail: props type %s is not declared among the *.go files in %s", typeName, dir)
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil, fmt.Errorf("gsxmail: %s in %s is not a type declaration", typeName, dir)
	}
	st, ok := tn.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("gsxmail: props type %s in %s is not a struct", typeName, dir)
	}

	qualifier := types.RelativeTo(rp.pkg)
	return propsFromStruct(typeName, st, qualifier, 1), nil
}

// propsFromStruct builds a Props from a resolved *types.Struct: st's own
// exported fields, classified against qualifier. depth bounds how many
// further levels of KindSlice-of-struct elements classifyField resolves
// into an ElemProps of their own (Resolve's top-level call passes 1: one
// level of <Each> loop-body nesting, matching the design spec's "props
// field paths, including nested reads" scope for v1 — a KindSlice field
// whose own element is a struct that itself declares a KindSlice field
// does not recurse further, avoiding unbounded work on a recursive type).
func propsFromStruct(name string, st *types.Struct, qualifier types.Qualifier, depth int) *Props {
	fields := make(map[string]Field, st.NumFields())
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		fields[f.Name()] = classifyField(f.Type(), qualifier, depth)
	}
	return &Props{Name: name, Fields: fields}
}

func isPlainIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func (r *Resolver) packageFor(dir string) *resolvedPackage {
	if rp, ok := r.cache[dir]; ok {
		return rp
	}
	rp := r.buildPackage(dir)
	r.cache[dir] = rp
	return rp
}

func (r *Resolver) buildPackage(dir string) *resolvedPackage {
	entries, err := fs.ReadDir(r.fsys, dir)
	if err != nil {
		return &resolvedPackage{err: fmt.Errorf("gsxmail: reading %s for props types: %w", dir, err)}
	}
	var files []*ast.File
	pkgName := ""
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		full := path.Join(dir, name)
		src, err := fs.ReadFile(r.fsys, full)
		if err != nil {
			return &resolvedPackage{err: fmt.Errorf("gsxmail: reading %s: %w", full, err)}
		}
		f, err := parser.ParseFile(r.fset, full, src, 0)
		if err != nil {
			return &resolvedPackage{err: fmt.Errorf("gsxmail: parsing %s for props types: %w", full, err)}
		}
		if pkgName == "" {
			pkgName = f.Name.Name
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return &resolvedPackage{err: fmt.Errorf("gsxmail: no *.go files in %s to resolve props types from", dir)}
	}

	var checkErrs []error
	conf := types.Config{
		Importer: importer.ForCompiler(r.fset, "source", nil),
		Error:    func(e error) { checkErrs = append(checkErrs, e) },
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	pkg, _ := conf.Check(pkgName, r.fset, files, info)
	if len(checkErrs) > 0 {
		return &resolvedPackage{err: fmt.Errorf("gsxmail: type-checking %s: %w", dir, checkErrs[0])}
	}
	return &resolvedPackage{pkg: pkg}
}

func classifyField(t types.Type, qualifier types.Qualifier, depth int) Field {
	kind, goType := classify(t, qualifier)
	f := Field{Kind: kind, GoType: goType}
	if kind == KindSlice {
		if elem, ok := sliceElem(t); ok {
			f.ElemKind, f.ElemType = classify(elem, qualifier)
			if depth > 0 {
				if elemSt, ok := elem.Underlying().(*types.Struct); ok {
					f.ElemProps = propsFromStruct(f.ElemType, elemSt, qualifier, depth-1)
				}
			}
		}
	}
	return f
}

func sliceElem(t types.Type) (types.Type, bool) {
	switch u := t.Underlying().(type) {
	case *types.Slice:
		return u.Elem(), true
	case *types.Array:
		return u.Elem(), true
	}
	return nil, false
}

func classify(t types.Type, qualifier types.Qualifier) (Kind, string) {
	goType := types.TypeString(t, qualifier)
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Info()&types.IsString != 0:
			return KindString, goType
		case u.Info()&types.IsInteger != 0:
			return KindInt, goType
		case u.Info()&types.IsFloat != 0:
			return KindFloat, goType
		case u.Info()&types.IsBoolean != 0:
			return KindBool, goType
		default:
			return KindOther, goType
		}
	case *types.Slice, *types.Array:
		return KindSlice, goType
	default:
		return KindOther, goType
	}
}

// BareFieldName reports whether src is exactly a bare props field read
// (propsName.Field, with no other operator), and if so, the field's name.
// EM013's message needs this fast path: a bare read of a non-scalar field
// gets the specific "has type %s; ... format structs before rendering"
// message, rather than the generic EM010 catch-all a nested occurrence
// would fall through to.
func BareFieldName(src, propsName string) (string, bool) {
	root, field, ok := ParseBareSelector(src)
	if !ok || field == "" || root != propsName {
		return "", false
	}
	return field, true
}

// ParseBareSelector parses src as a bare "root.field" reference, or a bare
// "root" identifier with no selector, and reports the two names. It takes
// no view on whether root is the props parameter or a loop binding — that
// judgment needs context (the props type, the active bindings) this parse
// step does not have; ResolveFieldPath and CheckSlicePath supply it. ok is
// false for anything else (a call, an index, a literal, and so on).
func ParseBareSelector(src string) (root, field string, ok bool) {
	fset := token.NewFileSet()
	expr, err := parser.ParseExprFrom(fset, "expr", src, 0)
	if err != nil {
		return "", "", false
	}
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		ident, ok := e.X.(*ast.Ident)
		if !ok {
			return "", "", false
		}
		return ident.Name, e.Sel.Name, true
	case *ast.Ident:
		return e.Name, "", true
	default:
		return "", "", false
	}
}

// ResolveFieldPath resolves a bare selector's root/field pair (as
// ParseBareSelector returns them) against propsName and bindings: root ==
// propsName reads props.field; root naming a bindings key reads that loop
// variable's own field (design spec section 6.1's "the binding
// participates in expression resolution inside the body"). field == ""
// (a bare loop-variable reference with no selector, such as <Each of=...
// as="tag"> then {tag}) resolves to the binding's own Kind, with no
// further Fields — a scalar-element loop has nothing to select from. ok is
// false when root names neither the props parameter nor a known binding,
// or when the named field does not exist on whichever side matched.
func ResolveFieldPath(root, field, propsName string, props *Props, bindings map[string]Binding) (Field, bool) {
	if root == propsName {
		if field == "" || props == nil {
			return Field{}, false
		}
		f, ok := props.Fields[field]
		return f, ok
	}
	b, ok := bindings[root]
	if !ok {
		return Field{}, false
	}
	if field == "" {
		return Field{Kind: b.Kind}, true
	}
	if b.Fields == nil {
		return Field{}, false
	}
	f, ok := b.Fields.Fields[field]
	return f, ok
}
