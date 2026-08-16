package lint

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"m31labs.dev/gosx/ir"
	"m31labs.dev/gsxmail/renderhtml"
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
// helpers (EM014/EM015), the caniemail matrix (EM101/EM102), and (design
// spec section 15, WP5.2) the Set's own Theme, which EM143 needs to judge
// whether a raw element's literal color has a dark-palette counterpart.
// The zero value selects DefaultMatrix(), treats every helper call as
// unregistered, and (Theme's own zero value carrying DarkMode "") never
// trips EM143, since that check only ever runs under DarkMode "adaptive".
type Options struct {
	Helpers map[string]any
	Matrix  *Matrix
	Theme   renderhtml.Theme
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
	w := &walker{
		file: file, prog: prog, matrix: matrix, helpers: opts.Helpers,
		theme: opts.Theme, darkTokens: collectDarkTokens(opts.Theme),
	}
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
	// theme and darkTokens back EM143 (design spec section 15, WP5.2): a
	// raw element's literal color with no counterpart in either palette,
	// checked only under DarkMode "adaptive" — see checkStyle.
	theme      renderhtml.Theme
	darkTokens map[string]bool
	diags      []Diagnostic
}

// exprContext is the props/loop-binding scope in effect while checking one
// component's expressions; <Each as="name"> extends bindings for its own
// children only.
type exprContext struct {
	templateName string
	propsName    string
	props        *typesafe.Props
	helpers      map[string]any
	bindings     map[string]typesafe.Binding
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
		bindings:     map[string]typesafe.Binding{},
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
		if v := typesafe.CheckInterpolation(n.Text, ctx.templateName, ctx.propsName, ctx.props, ctx.bindings, ctx.helpers); v != nil {
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
		if v := typesafe.CheckInterpolation(a.Expr, ctx.templateName, ctx.propsName, ctx.props, ctx.bindings, ctx.helpers); v != nil {
			w.add(n, v.Code, "error", v.Message)
		}
	}
}

// checkComponentAttr is checkElementAttr's narrower counterpart for
// email.* stdlib component attributes: it skips the raw-element-only rules
// (EM004 event attrs, EM104 class — no stdlib component accepts either),
// but still checks hrefs and every expression hole. slicePathAttrs names
// this component's own attributes (if any) that are slice-valued rather
// than interpolated text — <email.StatTable header={...}> and
// <email.StatRow cells={...}> (spec section 7.2's "Rows built from Each
// over props data or literal rows") — so checkComponentAttr can route them
// through CheckSlicePath instead of the scalar-only CheckInterpolation.
func (w *walker) checkComponentAttr(n *ir.Node, a ir.Attr, ctx *exprContext, slicePathAttrs map[string]bool) {
	if a.Name == "href" && a.Kind == ir.AttrStatic {
		w.checkHrefScheme(n, a.Value)
	}
	if a.Kind != ir.AttrExpr {
		return
	}
	if slicePathAttrs[a.Name] {
		label := n.Tag + " " + a.Name
		if _, v := typesafe.CheckSlicePath(label, a.Expr, ctx.templateName, ctx.propsName, ctx.props, ctx.bindings); v != nil {
			w.add(n, v.Code, "error", v.Message)
		}
		return
	}
	if v := typesafe.CheckInterpolation(a.Expr, ctx.templateName, ctx.propsName, ctx.props, ctx.bindings, ctx.helpers); v != nil {
		w.add(n, v.Code, "error", v.Message)
	}
}

// statTableSliceAttrs and statRowSliceAttrs name the one slice-valued
// attribute each of these two stdlib components accepts (the local name
// after the "email." namespace prefix splitTag strips): <email.StatTable
// header={...}> and <email.StatRow cells={...}> (design spec section 7.2).
// Every other email.* component's attributes are ordinary interpolated
// text or bool holes and stay on the generic CheckInterpolation path.
var statTableSliceAttrs = map[string]bool{"header": true}
var statRowSliceAttrs = map[string]bool{"cells": true}

// sliceAttrsFor names local (a "email." prefix already stripped) which of
// tag's own attributes are slice-valued, per statTableSliceAttrs and
// statRowSliceAttrs.
func sliceAttrsFor(local string) map[string]bool {
	switch local {
	case "StatTable":
		return statTableSliceAttrs
	case "StatRow":
		return statRowSliceAttrs
	default:
		return nil
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
// findAttr finds name in attrs and returns it, or nil if absent. checkShell
// (design spec section 15, WP5.2) is the one caller that needs the whole
// ir.Attr (kind and expression source, not just a resolved static value) —
// attrValue below serves every other caller's narrower need.
func findAttr(attrs []ir.Attr, name string) *ir.Attr {
	for i := range attrs {
		if attrs[i].Name == name {
			return &attrs[i]
		}
	}
	return nil
}

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
	w.checkDarkPaletteCoverage(n, raw)
}

// checkDarkPaletteCoverage implements EM143 (design spec section 15,
// WP5.2; pixel dossier section 5.3, rule 3's "Custom block color" case):
// under DarkMode "adaptive", a raw element's literal color or
// background-color that matches neither the light Theme's nor Theme.Dark's
// own tokens has nowhere to go when the adaptive style layer swaps in —
// a forced transform will recolor it unpredictably instead. It only
// judges "none" and "locked" themes' Custom colors — an author using
// their own theme's own tokens is always covered under "locked" — by
// staying a no-op there (darkTokens is only ever consulted here).
func (w *walker) checkDarkPaletteCoverage(n *ir.Node, raw string) {
	if w.theme.DarkStrategy() != "adaptive" {
		return
	}
	for _, value := range extractColorValues(raw) {
		hex := strings.ToUpper(strings.TrimSpace(value))
		if !isHexColor(hex) {
			continue // EM143 judges literal hex colors only; named colors and rgb()/var() are out of scope for a token-membership check
		}
		if !w.darkTokens[hex] {
			w.add(n, "EM143", "warn", fmt.Sprintf(
				"Custom block color %q has no dark-palette counterpart; forced transforms will recolor it unpredictably", value))
		}
	}
}

// extractColorValues re-splits raw the same way parseCSSDeclarations does,
// but keeps color and background-color's own values (parseCSSDeclarations
// keeps property names only, which EM101/EM102's matrix lookup is all
// EM103's caller needs).
func extractColorValues(raw string) map[string]string {
	out := map[string]string{}
	for _, decl := range strings.Split(raw, ";") {
		decl = strings.TrimSpace(decl)
		idx := strings.Index(decl, ":")
		if idx <= 0 {
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(decl[:idx]))
		if prop != "color" && prop != "background-color" {
			continue
		}
		out[prop] = strings.TrimSpace(decl[idx+1:])
	}
	return out
}

// isHexColor reports whether s (already upper-cased) is a six-digit "#RRGGBB" literal.
func isHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// collectDarkTokens gathers every color token from theme's light fields
// and (when set) theme.Dark, upper-cased, for EM143's membership check.
// It is cheap to build unconditionally; checkDarkPaletteCoverage is what
// gates its use to DarkMode "adaptive".
func collectDarkTokens(theme renderhtml.Theme) map[string]bool {
	tokens := map[string]bool{}
	add := func(hex string) {
		if hex != "" {
			tokens[strings.ToUpper(hex)] = true
		}
	}
	add(theme.ColorGround)
	add(theme.ColorCard)
	add(theme.ColorPanel)
	add(theme.ColorBorder)
	add(theme.ColorAccent)
	add(theme.ColorInk)
	add(theme.ColorBody)
	add(theme.ColorMuted)
	add(theme.ColorFaint)
	if theme.Dark != nil {
		add(theme.Dark.ColorGround)
		add(theme.Dark.ColorCard)
		add(theme.Dark.ColorPanel)
		add(theme.Dark.ColorBorder)
		add(theme.Dark.ColorAccent)
		add(theme.Dark.ColorInk)
		add(theme.Dark.ColorBody)
		add(theme.Dark.ColorMuted)
		add(theme.Dark.ColorFaint)
	}
	return tokens
}

// checkShell validates <email.Shell>'s own WP5.2 attributes (design spec
// section 15, WP5.2; pixel dossier section 4.2's Shell options and section
// 6.1's preheader contract):
//
//   - EM170 (warn) flags a Shell with no preheader attribute at all — a
//     discoverable authoring smell, not a rendering bug, since the source
//     shape (attribute present or not) is knowable without evaluating
//     anything.
//   - EM171 (error) rejects a literal preheader text over the emitted
//     block's 150-character pad target. A dynamic {expression} preheader
//     is not checked here — its length is not known until Render.
//   - EM172 (error) rejects an outlook attribute that is not a static "",
//     "ghost-tables", or "off": the per-Shell output-contract override is
//     a structural, compile-time choice, never a props-dependent one
//     (doc.Shell's own doc comment).
func (w *walker) checkShell(n *ir.Node) {
	preheaderAttr := findAttr(n.Attrs, "preheader")
	switch {
	case preheaderAttr == nil:
		w.add(n, "EM170", "warn", `Shell has no preheader; clients will preview the first body text instead`)
	case preheaderAttr.Kind == ir.AttrStatic:
		if count := utf8.RuneCountInString(preheaderAttr.Value); count > 150 {
			w.add(n, "EM171", "error", fmt.Sprintf(
				"preheader is %d characters; the limit is 150 (the emitted block pads to exactly 150)", count))
		}
	}

	outlookAttr := findAttr(n.Attrs, "outlook")
	if outlookAttr == nil {
		return
	}
	got := outlookAttr.Value
	valid := outlookAttr.Kind == ir.AttrStatic
	if valid {
		switch outlookAttr.Value {
		case "", "ghost-tables", "off":
		default:
			valid = false
		}
	} else {
		got = "{" + outlookAttr.Expr + "}"
	}
	if !valid {
		w.add(n, "EM172", "error", fmt.Sprintf(
			`<email.Shell> outlook attribute must be a static "", "ghost-tables", or "off"; got %q`, got))
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
	ns, local := splitTag(n.Tag)
	switch {
	case n.Tag == "If":
		w.checkIf(n, ctx)
	case n.Tag == "Each":
		w.checkEach(n, ctx)
	case n.Tag == "email.Columns":
		// Columns is special-cased ahead of the generic ns == "email"
		// branch below (WP5.3; pixel dossier section 4.9): it validates and
		// recurses into its own <email.Column> children itself, rather than
		// letting the generic per-node dispatch reach them — see
		// checkColumns and EM177's own doc comment for why.
		w.checkColumns(n, ctx)
	case ns == "email":
		// A recognized stdlib namespace. Which email.* member it names, and
		// whether that member exists at all, is package lower's job
		// (spec section 15, WP1); the lint only validates this node's own
		// attribute expressions and recurses into its children.
		slicePathAttrs := sliceAttrsFor(local)
		for _, a := range n.Attrs {
			w.checkComponentAttr(n, a, ctx, slicePathAttrs)
		}
		switch local {
		case "Shell":
			w.checkShell(n)
		case "Button":
			w.checkButton(n)
		case "Column":
			// A <email.Column> reached through this generic dispatch was
			// not consumed by checkColumns above, so it is not a direct
			// child of <email.Columns> (design spec section 15, WP5.3).
			w.add(n, "EM177", "error", `<email.Column> must be a direct child of <email.Columns>`)
		case "Hero":
			w.checkHero(n)
		case "Spacer":
			w.checkSpacer(n)
		case "Badge":
			w.checkBadge(n)
		}
		for _, c := range n.Children {
			w.checkNode(c, ctx)
		}
	default:
		w.add(n, "EM020", "error", fmt.Sprintf("component <%s> is not an email.* component and is not declared in this template set", n.Tag))
	}
}

// isBlankNode reports whether n is a whitespace-only text node, mirroring
// package lower's own isBlankText — the lint walker keeps its own copy
// rather than importing package lower, which itself would create an
// import cycle (lower already depends on nothing lint-shaped, and should
// stay that way).
func isBlankNode(n *ir.Node) bool {
	return n.Kind == ir.NodeText && strings.TrimSpace(n.Text) == ""
}

// checkButton implements EM175 (design spec section 15, WP5.3; pixel
// dossier section 4.4): <email.Button>'s variant attribute, when present,
// must be a static "primary", "secondary", or "link" — the button
// technique is a structural, compile-time choice, the same reasoning
// checkShell's own outlook attribute (EM172) already applies.
func (w *walker) checkButton(n *ir.Node) {
	a := findAttr(n.Attrs, "variant")
	if a == nil {
		return // defaults to "primary"
	}
	got := a.Value
	valid := a.Kind == ir.AttrStatic
	if valid {
		switch a.Value {
		case "", "primary", "secondary", "link":
		default:
			valid = false
		}
	} else {
		got = "{" + a.Expr + "}"
	}
	if !valid {
		w.add(n, "EM175", "error", fmt.Sprintf(
			`<email.Button> variant attribute must be a static "primary", "secondary", or "link"; got %q`, got))
	}
}

// checkColumns implements EM176 (design spec section 15, WP5.3; pixel
// dossier section 4.9): every child of <email.Columns> must be an
// <email.Column>, and there must be between two and four of them — narrow
// enough that each column's own max-width formula
// (renderhtml.writeColumns) stays legible on a 600px card, wide enough
// that "Columns" means something. checkColumns is reached before the
// generic per-node dispatch (checkComponentNode's own switch), so it
// handles its own children's attribute checks directly rather than
// recursing back through checkNode/checkComponentNode — a Column reached
// any other way trips EM177 instead (checkComponentNode's own "Column"
// case).
func (w *walker) checkColumns(n *ir.Node, ctx *exprContext) {
	var kids []*ir.Node
	for _, id := range n.Children {
		c := w.prog.NodeAt(id)
		if isBlankNode(c) {
			continue
		}
		if c.Kind != ir.NodeComponent || c.Tag != "email.Column" {
			w.add(n, "EM176", "error", fmt.Sprintf(`<email.Columns> children must be <email.Column>, got %q`, c.Tag))
			continue
		}
		kids = append(kids, c)
	}
	if len(kids) < 2 || len(kids) > 4 {
		w.add(n, "EM176", "error", fmt.Sprintf(
			`<email.Columns> must contain between 2 and 4 <email.Column> children, got %d`, len(kids)))
	}
	for _, c := range kids {
		for _, a := range c.Attrs {
			w.checkComponentAttr(c, a, ctx, nil)
		}
	}
}

// checkHero implements Hero's own attribute checks (design spec section
// 15, WP5.3; pixel dossier section 4.10). It reuses checkImg verbatim for
// src/alt — EM111 (an absolute https src) and EM112 (non-empty alt) apply
// to Hero exactly as they already do to a raw <img>, since checkImg reads
// n.Attrs generically regardless of whether n is an element or a
// component node. EM178 is the one WP5.3-specific addition: both width
// and height are required, the retina display-size attributes the pixel
// dossier's own retina rule (section 6.2) states.
func (w *walker) checkHero(n *ir.Node) {
	w.checkImg(n)
	for _, name := range []string{"width", "height"} {
		a := findAttr(n.Attrs, name)
		if a == nil || (a.Kind == ir.AttrStatic && strings.TrimSpace(a.Value) == "") {
			w.add(n, "EM178", "error", `<email.Hero> requires width and height attributes at display size; retina assets render at intrinsic size without them`)
			return
		}
	}
}

// checkSpacer implements EM179 (design spec section 15, WP5.3; pixel
// dossier section 4.8): <email.Spacer> requires a height attribute — the
// one thing that gives the spacer table its exact-height contract any
// meaning.
func (w *walker) checkSpacer(n *ir.Node) {
	a := findAttr(n.Attrs, "height")
	if a == nil || (a.Kind == ir.AttrStatic && strings.TrimSpace(a.Value) == "") {
		w.add(n, "EM179", "error", `<email.Spacer> requires a height attribute (a positive pixel integer)`)
	}
}

// checkBadge implements EM180 (design spec section 15, WP5.3; pixel
// dossier section 4.11): <email.Badge>'s tone attribute, when present,
// must be a static "neutral", "positive", "warning", or "critical" —
// renderhtml.badgeToneColor's own closed set.
func (w *walker) checkBadge(n *ir.Node) {
	a := findAttr(n.Attrs, "tone")
	if a == nil {
		return // defaults to "neutral"
	}
	got := a.Value
	valid := a.Kind == ir.AttrStatic
	if valid {
		switch a.Value {
		case "", "neutral", "positive", "warning", "critical":
		default:
			valid = false
		}
	} else {
		got = "{" + a.Expr + "}"
	}
	if !valid {
		w.add(n, "EM180", "error", fmt.Sprintf(
			`<email.Badge> tone attribute must be a static "neutral", "positive", "warning", or "critical"; got %q`, got))
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

	elemBinding := typesafe.Binding{Kind: typesafe.KindUnknown}
	switch {
	case ofAttr == nil || ofAttr.Kind != ir.AttrExpr:
		w.add(n, "EM032", "error", "<Each of={}> requires a slice or array props path; got the of attribute is required")
	default:
		if b, v := typesafe.CheckSlicePath("Each of", ofAttr.Expr, ctx.templateName, ctx.propsName, ctx.props, ctx.bindings); v != nil {
			w.add(n, v.Code, "error", v.Message)
		} else {
			elemBinding = b
		}
	}

	if asAttr == nil || asAttr.Kind != ir.AttrStatic || strings.TrimSpace(asAttr.Value) == "" {
		w.add(n, "EM033", "error", `<Each> requires as="name"; the binding name is the loop variable`)
	}

	childCtx := ctx
	if asAttr != nil && asAttr.Kind == ir.AttrStatic && strings.TrimSpace(asAttr.Value) != "" {
		bindings := make(map[string]typesafe.Binding, len(ctx.bindings)+1)
		for k, v := range ctx.bindings {
			bindings[k] = v
		}
		bindings[asAttr.Value] = elemBinding
		childCtx = &exprContext{
			templateName: ctx.templateName, propsName: ctx.propsName,
			props: ctx.props, helpers: ctx.helpers, bindings: bindings,
		}
	}
	for _, c := range n.Children {
		w.checkNode(c, childCtx)
	}
}
