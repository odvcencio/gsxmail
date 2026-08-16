package lint

import (
	"fmt"
	"net/url"
	"strings"

	"m31labs.dev/gosx/ir"
	"m31labs.dev/gsxmail/typesafe"
)

// elementAllowlist is the EM003 allowlist (design spec section 8): the
// only raw HTML elements an email template may use directly.
var elementAllowlist = map[string]bool{
	"table": true, "tr": true, "td": true, "th": true,
	"div": true, "span": true, "p": true, "a": true, "img": true,
	"br": true, "hr": true,
	"h1": true, "h2": true, "h3": true, "h4": true,
	"strong": true, "em": true,
	"ul": true, "ol": true, "li": true,
}

// Options configures CheckProgram beyond the compiled IR: the registered
// helpers (EM014/EM015) and the caniemail matrix (EM101/EM102). The zero
// value selects DefaultMatrix() and treats every helper call as
// unregistered.
type Options struct {
	Helpers map[string]any
	Matrix  *Matrix
}

// CheckProgram runs the email lint catalog, EM001 through EM112 (design
// spec section 8), over every component prog declares and returns every
// finding. file is recorded on every Diagnostic: gosx.Compile takes no
// filename, so ir.Span.File is always empty, and the caller (gsxmail.Load)
// is the one that knows which *.gsx path prog came from. resolver resolves
// each component's declared props struct via go/types; dir is that
// component's directory within resolver's underlying fs.FS. CheckProgram
// does not stop at the first finding: it collects every one, across every
// component prog declares, so a caller can fail closed with the complete
// list (spec section 7.1).
func CheckProgram(file string, prog *ir.Program, resolver *typesafe.Resolver, dir string, opts Options) []Diagnostic {
	matrix := opts.Matrix
	if matrix == nil {
		matrix = DefaultMatrix()
	}
	w := &walker{file: file, prog: prog, matrix: matrix, helpers: opts.Helpers}
	for _, comp := range prog.Components {
		w.checkComponent(comp, resolver, dir)
	}
	return w.diags
}

type walker struct {
	file    string
	prog    *ir.Program
	matrix  *Matrix
	helpers map[string]any
	diags   []Diagnostic
}

// exprContext is the props/loop-binding scope in effect while checking one
// component's expressions; <Each as="name"> extends bindings for its own
// children only.
type exprContext struct {
	templateName string
	propsName    string
	props        *typesafe.Props
	helpers      map[string]any
	bindings     map[string]typesafe.Kind
}

func (w *walker) add(n *ir.Node, code, severity, msg string) {
	w.diags = append(w.diags, Diagnostic{
		File: w.file, Line: n.Span.StartLine, Col: n.Span.StartCol,
		Code: code, Severity: severity, Message: msg,
	})
}

func (w *walker) checkComponent(comp ir.Component, resolver *typesafe.Resolver, dir string) {
	root := w.prog.NodeAt(comp.Root)
	if comp.IsIsland {
		w.add(root, "EM005", "error", `directive //gosx:island has no meaning in an email template; emails have no client runtime`)
	}
	if comp.IsEngine {
		w.add(root, "EM005", "error", `directive //gosx:engine has no meaning in an email template; emails have no client runtime`)
	}

	var props *typesafe.Props
	if resolver != nil && comp.PropsType != "" {
		// A resolution error (an undeclared or non-struct props type) is a
		// Go-source problem outside the email dialect: it is left for
		// gsxmail.Load's own compile-error path rather than an invented EM
		// code. props stays nil, so every props.field read below reports
		// EM012 instead — still fail-closed, just with the generic message.
		if p, err := resolver.Resolve(dir, comp.PropsType); err == nil {
			props = p
		}
	}

	propsName := comp.PropsName
	if propsName == "" {
		propsName = "props"
	}

	ctx := &exprContext{
		templateName: comp.Name,
		propsName:    propsName,
		props:        props,
		helpers:      w.helpers,
		bindings:     map[string]typesafe.Kind{},
	}
	w.checkNode(comp.Root, ctx)
}

func (w *walker) checkNode(id ir.NodeID, ctx *exprContext) {
	n := w.prog.NodeAt(id)
	switch n.Kind {
	case ir.NodeElement:
		w.checkElement(n, ctx)
	case ir.NodeComponent:
		w.checkComponentNode(n, ctx)
	case ir.NodeExpr:
		if v := typesafe.CheckInterpolation(n.Text, ctx.templateName, ctx.propsName, ctx.props, ctx.helpers); v != nil {
			w.add(n, v.Code, "error", v.Message)
		}
	case ir.NodeFragment:
		for _, c := range n.Children {
			w.checkNode(c, ctx)
		}
	case ir.NodeText, ir.NodeRawHTML:
		// Literal content, not a dialect expression: nothing to check.
	}
}

func (w *walker) checkElement(n *ir.Node, ctx *exprContext) {
	switch n.Tag {
	case "script":
		w.add(n, "EM001", "error", `element <script> is not allowed in an email template; mail clients do not run JavaScript`)
	case "form":
		w.add(n, "EM002", "error", `element <form> is not allowed in an email template; render an <email.CTA> link instead`)
	case "link":
		w.add(n, "EM006", "error", `<link rel="stylesheet"> is not allowed; email styles must be inline or in the shell <style> block`)
	default:
		if !elementAllowlist[n.Tag] {
			w.add(n, "EM003", "error", fmt.Sprintf("element <%s> is not on the email element allowlist", n.Tag))
		}
	}

	for _, a := range n.Attrs {
		w.checkElementAttr(n, a, ctx)
	}
	if n.Tag == "img" {
		w.checkImg(n)
	}
	for _, c := range n.Children {
		w.checkNode(c, ctx)
	}
}

func (w *walker) checkElementAttr(n *ir.Node, a ir.Attr, ctx *exprContext) {
	if isEventAttrName(a.Name) {
		w.add(n, "EM004", "error", fmt.Sprintf("attribute %s is an event handler; mail clients strip JavaScript", a.Name))
	}
	if a.Name == "class" {
		w.add(n, "EM104", "error", `attribute class is not supported on custom elements in v1; use style or an email.* component`)
	}
	if a.Name == "style" && a.Kind == ir.AttrStatic {
		w.checkStyle(n, a.Value)
	}
	if a.Name == "href" && a.Kind == ir.AttrStatic {
		w.checkHrefScheme(n, a.Value)
	}
	if a.Kind == ir.AttrExpr {
		if v := typesafe.CheckInterpolation(a.Expr, ctx.templateName, ctx.propsName, ctx.props, ctx.helpers); v != nil {
			w.add(n, v.Code, "error", v.Message)
		}
	}
}

// checkComponentAttr is checkElementAttr's narrower counterpart for
// email.* stdlib component attributes: it skips the raw-element-only rules
// (EM004 event attrs, EM104 class — no stdlib component accepts either),
// but still checks hrefs and every expression hole.
func (w *walker) checkComponentAttr(n *ir.Node, a ir.Attr, ctx *exprContext) {
	if a.Name == "href" && a.Kind == ir.AttrStatic {
		w.checkHrefScheme(n, a.Value)
	}
	if a.Kind == ir.AttrExpr {
		if v := typesafe.CheckInterpolation(a.Expr, ctx.templateName, ctx.propsName, ctx.props, ctx.helpers); v != nil {
			w.add(n, v.Code, "error", v.Message)
		}
	}
}

func isEventAttrName(name string) bool {
	return len(name) > 2 && strings.HasPrefix(strings.ToLower(name), "on")
}

func (w *walker) checkHrefScheme(n *ir.Node, raw string) {
	u, err := url.Parse(strings.TrimSpace(raw))
	scheme := ""
	if err == nil {
		scheme = strings.ToLower(u.Scheme)
	}
	switch scheme {
	case "https", "http", "mailto":
		return
	default:
		w.add(n, "EM110", "error", fmt.Sprintf("href scheme %q is not allowed; use https, http, or mailto", scheme))
	}
}

func (w *walker) checkImg(n *ir.Node) {
	src, hasSrc, srcStatic := attrValue(n.Attrs, "src")
	if !hasSrc || (srcStatic && !strings.HasPrefix(src, "https://")) {
		w.add(n, "EM111", "error", `img src must be an absolute https URL; mail clients cannot resolve relative paths`)
	}
	alt, hasAlt, altStatic := attrValue(n.Attrs, "alt")
	if !hasAlt || (altStatic && strings.TrimSpace(alt) == "") {
		w.add(n, "EM112", "error", `img requires non-empty alt text; image blocking is a common default`)
	}
}

// attrValue finds name in attrs and reports its static value (present is
// true and isStatic is true), that it is present but a dynamic expression
// gsxmail cannot check without rendering (present true, isStatic false), or
// that it is absent (present false).
func attrValue(attrs []ir.Attr, name string) (value string, present, isStatic bool) {
	for _, a := range attrs {
		if a.Name != name {
			continue
		}
		if a.Kind == ir.AttrStatic {
			return a.Value, true, true
		}
		return "", true, false
	}
	return "", false, false
}

func (w *walker) checkStyle(n *ir.Node, raw string) {
	decls, err := parseCSSDeclarations(raw)
	if err != nil {
		w.add(n, "EM103", "error", fmt.Sprintf("style attribute %q does not parse as CSS declarations", raw))
		return
	}
	for _, prop := range decls {
		for _, f := range w.matrix.Check(prop) {
			if f.Code == "n" {
				w.add(n, "EM101", "error", fmt.Sprintf(
					"style property %q is unsupported in %s (caniemail %s, snapshot %s); remove it or suppress with gsxmail:allow %q",
					prop, f.ClientLabel, SupportWord(f.Code), w.matrix.SnapshotDate(), prop))
			} else {
				w.add(n, "EM102", "warn", fmt.Sprintf(
					"style property %q has partial support in %s (caniemail %s); verify in preview",
					prop, f.ClientLabel, SupportWord(f.Code)))
			}
		}
	}
}

// parseCSSDeclarations splits raw on ";" into "property: value" pairs and
// returns each lowercased property name in source order. It is a minimal
// syntax check (EM103): balanced declarations with a non-empty name and
// value either side of one colon, and a property name that looks like a
// CSS identifier.
func parseCSSDeclarations(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var props []string
	for _, decl := range strings.Split(raw, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		idx := strings.Index(decl, ":")
		if idx <= 0 {
			return nil, fmt.Errorf("declaration %q has no property:value pair", decl)
		}
		prop := strings.ToLower(strings.TrimSpace(decl[:idx]))
		value := strings.TrimSpace(decl[idx+1:])
		if prop == "" || value == "" || !isCSSPropertyName(prop) {
			return nil, fmt.Errorf("declaration %q is not valid CSS", decl)
		}
		props = append(props, prop)
	}
	return props, nil
}

func isCSSPropertyName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		return false
	}
	return true
}

// splitTag splits a dotted tag such as "email.PanelRow" into its namespace
// and local name, mirroring package lower's private helper of the same
// name and shape.
func splitTag(tag string) (ns, local string) {
	if i := strings.LastIndex(tag, "."); i >= 0 {
		return tag[:i], tag[i+1:]
	}
	return "", tag
}

func (w *walker) checkComponentNode(n *ir.Node, ctx *exprContext) {
	ns, _ := splitTag(n.Tag)
	switch {
	case n.Tag == "If":
		w.checkIf(n, ctx)
	case n.Tag == "Each":
		w.checkEach(n, ctx)
	case ns == "email":
		// A recognized stdlib namespace. Which email.* member it names, and
		// whether that member exists at all, is package lower's job
		// (spec section 15, WP1); the lint only validates this node's own
		// attribute expressions and recurses into its children.
		for _, a := range n.Attrs {
			w.checkComponentAttr(n, a, ctx)
		}
		for _, c := range n.Children {
			w.checkNode(c, ctx)
		}
	default:
		w.add(n, "EM020", "error", fmt.Sprintf("component <%s> is not an email.* component and is not declared in this template set", n.Tag))
	}
}

func (w *walker) checkIf(n *ir.Node, ctx *exprContext) {
	var condAttr *ir.Attr
	for i := range n.Attrs {
		if n.Attrs[i].Name == "cond" {
			condAttr = &n.Attrs[i]
		}
	}
	valid := len(n.Attrs) == 1 && condAttr != nil && condAttr.Kind == ir.AttrExpr
	if valid {
		kind, v := typesafe.CheckExpr(condAttr.Expr, ctx.templateName, ctx.propsName, ctx.props, ctx.bindings, ctx.helpers)
		valid = v == nil && kind == typesafe.KindBool
	}
	if !valid {
		w.add(n, "EM030", "error", `<If> requires exactly one cond attribute with a bool expression`)
	}

	for _, c := range n.Children {
		if child := w.prog.NodeAt(c); child.Kind == ir.NodeText && strings.TrimSpace(child.Text) != "" {
			w.add(n, "EM031", "error", `<If> child is bare text; wrap it in an element so the text twin can place it`)
		}
		w.checkNode(c, ctx)
	}
}

func (w *walker) checkEach(n *ir.Node, ctx *exprContext) {
	var ofAttr, asAttr *ir.Attr
	for i := range n.Attrs {
		switch n.Attrs[i].Name {
		case "of":
			ofAttr = &n.Attrs[i]
		case "as":
			asAttr = &n.Attrs[i]
		}
	}

	elemKind := typesafe.KindUnknown
	switch {
	case ofAttr == nil || ofAttr.Kind != ir.AttrExpr:
		w.add(n, "EM032", "error", "<Each of={}> requires a slice or array props path; got the of attribute is required")
	default:
		field, isBareField := typesafe.BareFieldName(ofAttr.Expr, ctx.propsName)
		switch {
		case !isBareField:
			w.add(n, "EM032", "error", fmt.Sprintf("<Each of={%s}> requires a slice or array props path; got a non-props-path expression", ofAttr.Expr))
		case ctx.props == nil:
			w.add(n, "EM012", "error", fmt.Sprintf("template %s reads props.%s but type %s has no field %s", ctx.templateName, field, "<no props type>", field))
		default:
			f, exists := ctx.props.Fields[field]
			switch {
			case !exists:
				w.add(n, "EM012", "error", fmt.Sprintf("template %s reads props.%s but type %s has no field %s", ctx.templateName, field, ctx.props.Name, field))
			case f.Kind != typesafe.KindSlice:
				w.add(n, "EM032", "error", fmt.Sprintf("<Each of={%s}> requires a slice or array props path; got %s", ofAttr.Expr, f.GoType))
			default:
				elemKind = f.ElemKind
			}
		}
	}

	if asAttr == nil || asAttr.Kind != ir.AttrStatic || strings.TrimSpace(asAttr.Value) == "" {
		w.add(n, "EM033", "error", `<Each> requires as="name"; the binding name is the loop variable`)
	}

	childCtx := ctx
	if asAttr != nil && asAttr.Kind == ir.AttrStatic && strings.TrimSpace(asAttr.Value) != "" {
		bindings := make(map[string]typesafe.Kind, len(ctx.bindings)+1)
		for k, v := range ctx.bindings {
			bindings[k] = v
		}
		bindings[asAttr.Value] = elemKind
		childCtx = &exprContext{
			templateName: ctx.templateName, propsName: ctx.propsName,
			props: ctx.props, helpers: ctx.helpers, bindings: bindings,
		}
	}
	for _, c := range n.Children {
		w.checkNode(c, childCtx)
	}
}
