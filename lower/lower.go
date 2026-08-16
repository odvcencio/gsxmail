// Package lower converts a compiled gosx ir.Program into a gsxmail
// doc.EmailDoc: it resolves email.* stdlib tags, the <Each>/<If> builtins,
// and a raw-element Custom subtree escape hatch, inlining every attribute
// expression into a doc.Expr or doc.FieldPath value. This is pipeline
// stage 2 (link) and stage 3 (lower to EmailDoc) from the design spec,
// section 7.2. Lower runs only after gsxmail.Load's lint pass has already
// cleared the same program (design spec section 7.1), so it is not itself
// a fail-closed check-time gate: it trusts the shapes lint already
// validated (a <If> has exactly one bool cond, an <Each> has of and as,
// ...) and stays defensive, not exhaustive, about anything else.
package lower

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"m31labs.dev/gosx/ir"
	"m31labs.dev/gsxmail/doc"
)

// Lower finds the component named name in prog and lowers it to an
// EmailDoc. The component's root node must be an <email.Shell> element.
func Lower(prog *ir.Program, name string) (*doc.EmailDoc, error) {
	comp, ok := findComponent(prog, name)
	if !ok {
		return nil, fmt.Errorf("gsxmail: template %q not found", name)
	}
	root := prog.NodeAt(comp.Root)
	ns, local := splitTag(root.Tag)
	if root.Kind != ir.NodeComponent || ns != "email" || local != "Shell" {
		return nil, fmt.Errorf("gsxmail: template %q must return <email.Shell>...</email.Shell> at its root, got %q", name, root.Tag)
	}

	propsName := comp.PropsName
	if propsName == "" {
		propsName = "props"
	}

	shell, err := lowerShell(root, propsName)
	if err != nil {
		return nil, fmt.Errorf("gsxmail: template %q: %w", name, err)
	}
	blocks, err := lowerBlocks(prog, root.Children, propsName, nil)
	if err != nil {
		return nil, fmt.Errorf("gsxmail: template %q: %w", name, err)
	}
	return &doc.EmailDoc{Shell: shell, Blocks: blocks}, nil
}

func findComponent(prog *ir.Program, name string) (ir.Component, bool) {
	for _, c := range prog.Components {
		if c.Name == name {
			return c, true
		}
	}
	return ir.Component{}, false
}

// splitTag splits a dotted tag such as "email.PanelRow" into its namespace
// and local name. A bare tag ("Shell") splits to ("", "Shell").
func splitTag(tag string) (ns, local string) {
	if i := strings.LastIndex(tag, "."); i >= 0 {
		return tag[:i], tag[i+1:]
	}
	return "", tag
}

func lowerShell(root *ir.Node, propsName string) (doc.Shell, error) {
	var s doc.Shell
	var err error
	if s.Wordmark, err = attrExpr(root.Attrs, "wordmark", propsName, nil); err != nil {
		return s, err
	}
	if s.ShortCode, err = attrExpr(root.Attrs, "shortCode", propsName, nil); err != nil {
		return s, err
	}
	if s.Tagline, err = attrExpr(root.Attrs, "tagline", propsName, nil); err != nil {
		return s, err
	}
	if s.Title, err = attrExpr(root.Attrs, "title", propsName, nil); err != nil {
		return s, err
	}
	if s.Lang, err = attrExpr(root.Attrs, "lang", propsName, nil); err != nil {
		return s, err
	}
	return s, nil
}

// lowerBlocks walks children (a Shell's, an <Each>'s, or an <If>'s
// children in source order), skipping whitespace-only text nodes the gosx
// parser inserts between JSX-style element siblings, and lowers every
// child to a doc.Block: an email.* component, the <Each>/<If> builtins, or
// (design spec section 7.2's Custom escape hatch) a raw element subtree.
// bindings names the loop variables currently in scope (nil outside any
// <Each>).
func lowerBlocks(prog *ir.Program, children []ir.NodeID, propsName string, bindings map[string]bool) ([]doc.Block, error) {
	var blocks []doc.Block
	for _, id := range children {
		n := prog.NodeAt(id)
		if isBlankText(n) {
			continue
		}
		switch {
		case n.Kind == ir.NodeComponent && n.Tag == "Each":
			each, err := lowerEach(prog, n, propsName, bindings)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, each)
		case n.Kind == ir.NodeComponent && n.Tag == "If":
			ifBlock, err := lowerIf(prog, n, propsName, bindings)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, ifBlock)
		case n.Kind == ir.NodeComponent:
			ns, local := splitTag(n.Tag)
			if ns != "email" {
				return nil, fmt.Errorf("component <%s> is not an email.* component and is not declared in this template set", n.Tag)
			}
			block, err := lowerBlock(prog, local, n, propsName, bindings)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		case n.Kind == ir.NodeElement:
			root, err := lowerCustom(prog, n, propsName, bindings)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, doc.Custom{Root: root})
		default:
			return nil, fmt.Errorf("unexpected %s here; only email.* components, <Each>, <If>, and a raw element (Custom) are allowed", kindName(n.Kind))
		}
	}
	return blocks, nil
}

func lowerBlock(prog *ir.Program, local string, n *ir.Node, propsName string, bindings map[string]bool) (doc.Block, error) {
	switch local {
	case "Signal":
		text, err := attrExpr(n.Attrs, "text", propsName, bindings)
		if err != nil {
			return nil, err
		}
		return doc.Signal{Text: text}, nil
	case "Headline":
		title, err := attrExpr(n.Attrs, "title", propsName, bindings)
		if err != nil {
			return nil, err
		}
		lede, err := attrExpr(n.Attrs, "lede", propsName, bindings)
		if err != nil {
			return nil, err
		}
		return doc.Headline{Title: title, Lede: lede}, nil
	case "Panel":
		rows, err := lowerPanelRows(prog, n.Children, propsName, bindings)
		if err != nil {
			return nil, err
		}
		return doc.Panel{Rows: rows}, nil
	case "CTA":
		label, err := attrExpr(n.Attrs, "label", propsName, bindings)
		if err != nil {
			return nil, err
		}
		href, err := attrExpr(n.Attrs, "href", propsName, bindings)
		if err != nil {
			return nil, err
		}
		return doc.CTA{Label: label, Href: href}, nil
	case "PickList":
		title, err := attrExpr(n.Attrs, "title", propsName, bindings)
		if err != nil {
			return nil, err
		}
		items, err := lowerItems(prog, n.Children, propsName, bindings)
		if err != nil {
			return nil, err
		}
		return doc.PickList{Title: title, Items: items}, nil
	case "Footer":
		signoff, err := attrExpr(n.Attrs, "signoff", propsName, bindings)
		if err != nil {
			return nil, err
		}
		note, err := attrExpr(n.Attrs, "note", propsName, bindings)
		if err != nil {
			return nil, err
		}
		return doc.Footer{Signoff: signoff, Note: note}, nil
	case "Note":
		text, err := attrExpr(n.Attrs, "text", propsName, bindings)
		if err != nil {
			return nil, err
		}
		return doc.Note{Text: text}, nil
	case "Divider":
		return doc.Divider{}, nil
	case "StatTable":
		return lowerStatTable(prog, n, propsName, bindings)
	default:
		return nil, fmt.Errorf("component <email.%s> is not on the email.* component list (Shell, Signal, Headline, Panel/PanelRow, CTA, PickList/Item, Footer, Note, Divider, StatTable/StatRow)", local)
	}
}

func lowerPanelRows(prog *ir.Program, children []ir.NodeID, propsName string, bindings map[string]bool) ([]doc.PanelRow, error) {
	var rows []doc.PanelRow
	for _, id := range children {
		n := prog.NodeAt(id)
		if isBlankText(n) {
			continue
		}
		ns, local := splitTag(n.Tag)
		if n.Kind != ir.NodeComponent || ns != "email" || local != "PanelRow" {
			return nil, fmt.Errorf("<email.Panel> children must be <email.PanelRow>, got %q", n.Tag)
		}
		label, err := attrExpr(n.Attrs, "label", propsName, bindings)
		if err != nil {
			return nil, err
		}
		value, err := attrExpr(n.Attrs, "value", propsName, bindings)
		if err != nil {
			return nil, err
		}
		rows = append(rows, doc.PanelRow{Label: label, Value: value})
	}
	return rows, nil
}

func lowerItems(prog *ir.Program, children []ir.NodeID, propsName string, bindings map[string]bool) ([]doc.Item, error) {
	var items []doc.Item
	for _, id := range children {
		n := prog.NodeAt(id)
		if isBlankText(n) {
			continue
		}
		ns, local := splitTag(n.Tag)
		if n.Kind != ir.NodeComponent || ns != "email" || local != "Item" {
			return nil, fmt.Errorf("<email.PickList> children must be <email.Item>, got %q", n.Tag)
		}
		text, err := inlineContent(prog, n.Children, propsName, bindings)
		if err != nil {
			return nil, err
		}
		items = append(items, doc.Item{Text: text})
	}
	return items, nil
}

// lowerEach lowers <Each of={...} as="name">...</Each>: a top-level
// builtin that expands to zero or more sibling Blocks at Resolve time, one
// per element of the slice at Of (design spec section 6.5).
func lowerEach(prog *ir.Program, n *ir.Node, propsName string, bindings map[string]bool) (doc.Each, error) {
	of, err := parseFieldPathAttr(n.Attrs, "of", propsName, bindings)
	if err != nil {
		return doc.Each{}, err
	}
	as := staticAttrValue(n.Attrs, "as")
	if as == "" {
		return doc.Each{}, fmt.Errorf("<Each> requires as=\"name\"; the binding name is the loop variable")
	}
	body, err := lowerBlocks(prog, n.Children, propsName, extendBindings(bindings, as))
	if err != nil {
		return doc.Each{}, err
	}
	return doc.Each{Of: of, As: as, Body: body}, nil
}

// lowerIf lowers <If cond={...}>...</If>: Body resolves only when Cond
// evaluates true (design spec section 6.5). EM030/EM031 (lint) already
// guarantee cond is exactly one bool expression and every child is an
// element, never bare text.
func lowerIf(prog *ir.Program, n *ir.Node, propsName string, bindings map[string]bool) (doc.If, error) {
	condAttr := findAttr(n.Attrs, "cond")
	if condAttr == nil || condAttr.Kind != ir.AttrExpr {
		return doc.If{}, fmt.Errorf("<If> requires exactly one cond attribute with a bool expression")
	}
	cond, err := parseExprSource(condAttr.Expr, propsName, bindings)
	if err != nil {
		return doc.If{}, err
	}
	body, err := lowerBlocks(prog, n.Children, propsName, bindings)
	if err != nil {
		return doc.If{}, err
	}
	return doc.If{Cond: cond, Body: body}, nil
}

// lowerStatTable lowers <email.StatTable>: its title, its optional
// slice-valued header attribute, and its rows — a mix of literal
// <email.StatRow> children and <Each>-wrapped ones, in source order
// (design spec section 6.5's worked example).
func lowerStatTable(prog *ir.Program, n *ir.Node, propsName string, bindings map[string]bool) (doc.Block, error) {
	title, err := attrExpr(n.Attrs, "title", propsName, bindings)
	if err != nil {
		return nil, err
	}
	header, err := parseFieldPathAttr(n.Attrs, "header", propsName, bindings)
	if err != nil {
		return nil, err
	}

	var rows []doc.RowSpec
	for _, id := range n.Children {
		c := prog.NodeAt(id)
		if isBlankText(c) {
			continue
		}
		switch {
		case c.Kind == ir.NodeComponent && c.Tag == "email.StatRow":
			row, err := lowerStatRow(c, propsName, bindings)
			if err != nil {
				return nil, err
			}
			rows = append(rows, doc.RowSpec{Literal: &row})
		case c.Kind == ir.NodeComponent && c.Tag == "Each":
			of, err := parseFieldPathAttr(c.Attrs, "of", propsName, bindings)
			if err != nil {
				return nil, err
			}
			as := staticAttrValue(c.Attrs, "as")
			if as == "" {
				return nil, fmt.Errorf("<Each> requires as=\"name\"; the binding name is the loop variable")
			}
			rowNode, err := singleStatRowChild(prog, c)
			if err != nil {
				return nil, err
			}
			row, err := lowerStatRow(rowNode, propsName, extendBindings(bindings, as))
			if err != nil {
				return nil, err
			}
			rows = append(rows, doc.RowSpec{Each: &doc.EachRows{Of: of, As: as, Row: row}})
		default:
			return nil, fmt.Errorf("<email.StatTable> children must be <email.StatRow> or <Each>, got %q", c.Tag)
		}
	}
	return doc.StatTable{Title: title, Header: header, Rows: rows}, nil
}

// singleStatRowChild finds the one <email.StatRow> an <Each> inside an
// <email.StatTable> must wrap (design spec section 6.5's worked example
// has exactly this shape).
func singleStatRowChild(prog *ir.Program, each *ir.Node) (*ir.Node, error) {
	var found *ir.Node
	for _, id := range each.Children {
		c := prog.NodeAt(id)
		if isBlankText(c) {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("<Each> inside <email.StatTable> must wrap exactly one <email.StatRow>")
		}
		if c.Kind != ir.NodeComponent || c.Tag != "email.StatRow" {
			return nil, fmt.Errorf("<Each> inside <email.StatTable> must wrap <email.StatRow>, got %q", kindName(c.Kind))
		}
		found = c
	}
	if found == nil {
		return nil, fmt.Errorf("<Each> inside <email.StatTable> has no <email.StatRow> child")
	}
	return found, nil
}

func lowerStatRow(n *ir.Node, propsName string, bindings map[string]bool) (doc.StatRow, error) {
	cells, err := parseFieldPathAttr(n.Attrs, "cells", propsName, bindings)
	if err != nil {
		return doc.StatRow{}, err
	}
	if cells.IsZero() {
		return doc.StatRow{}, fmt.Errorf("<email.StatRow> requires a cells={} attribute")
	}
	mark, err := boolAttrExpr(n.Attrs, "mark", propsName, bindings)
	if err != nil {
		return doc.StatRow{}, err
	}
	return doc.StatRow{Cells: cells, Mark: mark}, nil
}

// lowerCustom lowers one raw element into a doc.CustomNode (design spec
// section 7.2's Custom pass-through). It only ever runs on a subtree the
// email lint already cleared (EM001-EM006 disallowed elements, EM101-EM104
// style/class rules, EM110-EM112 href/img rules), so it does not
// re-validate any of that — it only needs to carry the subtree's shape and
// expressions through to render time.
func lowerCustom(prog *ir.Program, n *ir.Node, propsName string, bindings map[string]bool) (doc.CustomNode, error) {
	attrs := make([]doc.CustomAttr, 0, len(n.Attrs))
	for _, a := range n.Attrs {
		switch a.Kind {
		case ir.AttrStatic:
			attrs = append(attrs, doc.CustomAttr{Name: a.Name, Value: doc.Static(a.Value)})
		case ir.AttrExpr:
			e, err := parseExprSource(a.Expr, propsName, bindings)
			if err != nil {
				return doc.CustomNode{}, err
			}
			attrs = append(attrs, doc.CustomAttr{Name: a.Name, Value: e})
		case ir.AttrBool:
			attrs = append(attrs, doc.CustomAttr{Name: a.Name, Value: doc.Static("")})
		default:
			return doc.CustomNode{}, fmt.Errorf("attribute %s: only static, {expression}, and bool values are supported inside a Custom subtree", a.Name)
		}
	}
	children, err := lowerCustomChildren(prog, n.Children, propsName, bindings)
	if err != nil {
		return doc.CustomNode{}, err
	}
	return doc.CustomNode{Tag: n.Tag, Attrs: attrs, Children: children}, nil
}

func lowerCustomChildren(prog *ir.Program, ids []ir.NodeID, propsName string, bindings map[string]bool) ([]doc.CustomNode, error) {
	var out []doc.CustomNode
	for _, id := range ids {
		n := prog.NodeAt(id)
		switch n.Kind {
		case ir.NodeText:
			if strings.TrimSpace(n.Text) == "" {
				continue
			}
			out = append(out, doc.CustomNode{IsText: true, Text: doc.Static(n.Text)})
		case ir.NodeExpr:
			e, err := parseExprSource(n.Text, propsName, bindings)
			if err != nil {
				return nil, err
			}
			out = append(out, doc.CustomNode{IsText: true, Text: e})
		case ir.NodeElement:
			child, err := lowerCustom(prog, n, propsName, bindings)
			if err != nil {
				return nil, err
			}
			out = append(out, child)
		case ir.NodeFragment:
			flattened, err := lowerCustomChildren(prog, n.Children, propsName, bindings)
			if err != nil {
				return nil, err
			}
			out = append(out, flattened...)
		default:
			return nil, fmt.Errorf("a Custom subtree may only contain raw elements and text/expression content; %s is not allowed inside one", kindName(n.Kind))
		}
	}
	return out, nil
}

// inlineContent concatenates an element's text and expression children, in
// source order, into one Expr. Whitespace-only text nodes are dropped.
func inlineContent(prog *ir.Program, children []ir.NodeID, propsName string, bindings map[string]bool) (doc.Expr, error) {
	var parts []doc.Expr
	for _, id := range children {
		n := prog.NodeAt(id)
		switch n.Kind {
		case ir.NodeText:
			if strings.TrimSpace(n.Text) == "" {
				continue
			}
			parts = append(parts, doc.Expr{Kind: doc.ExprLiteralString, Literal: n.Text})
		case ir.NodeExpr:
			e, err := parseExprSource(n.Text, propsName, bindings)
			if err != nil {
				return doc.Expr{}, err
			}
			parts = append(parts, flattenConcat(e)...)
		default:
			return doc.Expr{}, fmt.Errorf("<email.Item> content must be text or {expression}, got %s", kindName(n.Kind))
		}
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return doc.Expr{Kind: doc.ExprConcat, Parts: parts}, nil
}

func isBlankText(n *ir.Node) bool {
	return n.Kind == ir.NodeText && strings.TrimSpace(n.Text) == ""
}

func kindName(k ir.NodeKind) string {
	switch k {
	case ir.NodeElement:
		return "element"
	case ir.NodeComponent:
		return "component"
	case ir.NodeText:
		return "text"
	case ir.NodeExpr:
		return "expression"
	case ir.NodeFragment:
		return "fragment"
	case ir.NodeRawHTML:
		return "raw HTML"
	default:
		return "node"
	}
}

func findAttr(attrs []ir.Attr, name string) *ir.Attr {
	for i := range attrs {
		if attrs[i].Name == name {
			return &attrs[i]
		}
	}
	return nil
}

func staticAttrValue(attrs []ir.Attr, name string) string {
	a := findAttr(attrs, name)
	if a == nil || a.Kind != ir.AttrStatic {
		return ""
	}
	return a.Value
}

// extendBindings returns a new binding set with name added, leaving
// bindings itself untouched (so a sibling <Each> at the same nesting level
// never sees name).
func extendBindings(bindings map[string]bool, name string) map[string]bool {
	out := make(map[string]bool, len(bindings)+1)
	for k := range bindings {
		out[k] = true
	}
	out[name] = true
	return out
}

// attrExpr finds the named attribute on attrs and compiles its value to an
// Expr. A missing attribute lowers to an empty static Expr rather than an
// error, matching every prior release: a missing required attribute
// surfaces as an empty rendered field rather than a lower-time panic — the
// email lint (EM-catalog, section 8) is what actually enforces which
// attributes are required.
func attrExpr(attrs []ir.Attr, name, propsName string, bindings map[string]bool) (doc.Expr, error) {
	a := findAttr(attrs, name)
	if a == nil {
		return doc.Expr{}, nil
	}
	switch a.Kind {
	case ir.AttrStatic:
		return doc.Static(a.Value), nil
	case ir.AttrExpr:
		return parseExprSource(a.Expr, propsName, bindings)
	default:
		return doc.Expr{}, fmt.Errorf("attribute %s: only static and {expression} values are supported in v1", name)
	}
}

// boolAttrExpr is attrExpr for an optional bool-context attribute
// (StatRow's mark): a missing attribute defaults to the literal false,
// rather than the empty-string sentinel attrExpr's text-context callers
// use, since resolveBool has no analogous "empty string means false" rule.
func boolAttrExpr(attrs []ir.Attr, name, propsName string, bindings map[string]bool) (doc.Expr, error) {
	a := findAttr(attrs, name)
	if a == nil {
		return doc.Expr{Kind: doc.ExprLiteralBool, Bool: false}, nil
	}
	switch a.Kind {
	case ir.AttrExpr:
		return parseExprSource(a.Expr, propsName, bindings)
	case ir.AttrBool:
		return doc.Expr{Kind: doc.ExprLiteralBool, Bool: true}, nil
	default:
		return doc.Expr{}, fmt.Errorf("attribute %s: expected a {bool expression} value", name)
	}
}

// parseFieldPathAttr finds the named attribute and compiles it as a bare
// slice-valued field path (<Each of=...>, StatTable's header, StatRow's
// cells). A missing attribute returns the zero FieldPath (StatTable.Header
// treats that as "no header row"; callers that require the attribute,
// such as StatRow's cells, check IsZero themselves).
func parseFieldPathAttr(attrs []ir.Attr, name, propsName string, bindings map[string]bool) (doc.FieldPath, error) {
	a := findAttr(attrs, name)
	if a == nil {
		return doc.FieldPath{}, nil
	}
	if a.Kind != ir.AttrExpr {
		return doc.FieldPath{}, fmt.Errorf("attribute %s: requires a {props.field} or {binding.field} expression", name)
	}
	return parseFieldPath(a.Expr, propsName, bindings)
}

// parseFieldPath compiles src as a bare "root.field" (or bare "root")
// reference, matching typesafe.ParseBareSelector/ResolveFieldPath's same
// grammar (the lint already required this shape before Lower ever runs).
func parseFieldPath(src, propsName string, bindings map[string]bool) (doc.FieldPath, error) {
	expr, err := parser.ParseExpr(src)
	if err != nil {
		return doc.FieldPath{}, fmt.Errorf("expression %q does not parse as a Go expression: %w", src, err)
	}
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		ident, ok := e.X.(*ast.Ident)
		if !ok {
			return doc.FieldPath{}, fmt.Errorf("expression %q is not a bare field path", src)
		}
		if ident.Name == propsName {
			return doc.FieldPath{Field: e.Sel.Name}, nil
		}
		if bindings[ident.Name] {
			return doc.FieldPath{Binding: ident.Name, Field: e.Sel.Name}, nil
		}
		return doc.FieldPath{}, fmt.Errorf("expression %q reads a field of %q, which is not the props parameter or a known loop binding", src, ident.Name)
	case *ast.Ident:
		if bindings[e.Name] {
			return doc.FieldPath{Binding: e.Name}, nil
		}
		return doc.FieldPath{}, fmt.Errorf("expression %q is not a props field or a known loop binding", src)
	default:
		return doc.FieldPath{}, fmt.Errorf("expression %q is not a bare field path", src)
	}
}

// parseExprSource compiles a raw Go expression source string (as recorded
// on ir.Attr.Expr / ir.Node.Text for NodeExpr) into a doc.Expr, following
// the email dialect's grammar (spec section 6.1): string/int/float/bool
// literals, direct props or loop-binding field reads, string +
// concatenation, comparisons, !, len(), and helper calls. It builds the
// same tree shape typesafe/expr.go's checker already proved src belongs to
// — parseExprSource does not re-derive that proof; it trusts it.
func parseExprSource(src, propsName string, bindings map[string]bool) (doc.Expr, error) {
	expr, err := parser.ParseExpr(src)
	if err != nil {
		return doc.Expr{}, fmt.Errorf("expression %q does not parse as a Go expression: %w", src, err)
	}
	e, err := buildExpr(expr, propsName, bindings)
	if err != nil {
		return doc.Expr{}, fmt.Errorf("expression %q: %w", src, err)
	}
	return e, nil
}

func buildExpr(n ast.Expr, propsName string, bindings map[string]bool) (doc.Expr, error) {
	switch e := n.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.STRING:
			s, err := strconv.Unquote(e.Value)
			if err != nil {
				return doc.Expr{}, fmt.Errorf("cannot unquote string literal %s: %w", e.Value, err)
			}
			return doc.Expr{Kind: doc.ExprLiteralString, Literal: s}, nil
		case token.INT, token.FLOAT:
			return doc.Expr{Kind: doc.ExprLiteralNumber, Literal: e.Value}, nil
		default:
			return doc.Expr{}, fmt.Errorf("literal %s is not a string, int, or float; the email dialect allows only those in v1", e.Value)
		}
	case *ast.Ident:
		switch e.Name {
		case "true":
			return doc.Expr{Kind: doc.ExprLiteralBool, Bool: true}, nil
		case "false":
			return doc.Expr{Kind: doc.ExprLiteralBool, Bool: false}, nil
		}
		if bindings[e.Name] {
			return doc.Expr{Kind: doc.ExprField, Binding: e.Name}, nil
		}
		return doc.Expr{}, fmt.Errorf("identifier %q is not true, false, or a known loop binding", e.Name)
	case *ast.SelectorExpr:
		ident, ok := e.X.(*ast.Ident)
		if !ok {
			return doc.Expr{}, fmt.Errorf("is not in the email dialect; allowed: %s.<field>, a loop binding's field, string literals, and string +", propsName)
		}
		if ident.Name == propsName {
			return doc.Expr{Kind: doc.ExprField, Field: e.Sel.Name}, nil
		}
		if bindings[ident.Name] {
			return doc.Expr{Kind: doc.ExprField, Binding: ident.Name, Field: e.Sel.Name}, nil
		}
		return doc.Expr{}, fmt.Errorf("is not in the email dialect; allowed: %s.<field>, a loop binding's field, string literals, and string +", propsName)
	case *ast.BinaryExpr:
		left, err := buildExpr(e.X, propsName, bindings)
		if err != nil {
			return doc.Expr{}, err
		}
		right, err := buildExpr(e.Y, propsName, bindings)
		if err != nil {
			return doc.Expr{}, err
		}
		switch e.Op {
		case token.ADD:
			parts := append(flattenConcat(left), flattenConcat(right)...)
			return doc.Expr{Kind: doc.ExprConcat, Parts: parts}, nil
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
			return doc.Expr{Kind: doc.ExprCompare, Op: opString(e.Op), Left: &left, Right: &right}, nil
		default:
			return doc.Expr{}, fmt.Errorf("operator %q is not in the email dialect; only string +, ==, !=, <, <=, >, and >= are allowed in v1", e.Op)
		}
	case *ast.UnaryExpr:
		if e.Op != token.NOT {
			return doc.Expr{}, fmt.Errorf("operator %q is not in the email dialect; only ! is allowed in v1", e.Op)
		}
		x, err := buildExpr(e.X, propsName, bindings)
		if err != nil {
			return doc.Expr{}, err
		}
		return doc.Expr{Kind: doc.ExprNot, X: &x}, nil
	case *ast.ParenExpr:
		return buildExpr(e.X, propsName, bindings)
	case *ast.CallExpr:
		fn, ok := e.Fun.(*ast.Ident)
		if !ok {
			return doc.Expr{}, fmt.Errorf("is not in the email dialect; only len() and a registered helper may be called")
		}
		args := make([]doc.Expr, len(e.Args))
		for i, a := range e.Args {
			ae, err := buildExpr(a, propsName, bindings)
			if err != nil {
				return doc.Expr{}, err
			}
			args[i] = ae
		}
		if fn.Name == "len" {
			if len(args) != 1 {
				return doc.Expr{}, fmt.Errorf("len() takes exactly one argument")
			}
			return doc.Expr{Kind: doc.ExprLen, X: &args[0]}, nil
		}
		return doc.Expr{Kind: doc.ExprCall, Call: fn.Name, Args: args}, nil
	default:
		return doc.Expr{}, fmt.Errorf("is not in the email dialect; allowed: props/binding field paths, literals, string +, comparisons, len(), registered helpers")
	}
}

// flattenConcat returns e's own Parts when e is itself an ExprConcat
// (already built from a nested "+" chain), or []Expr{e} otherwise, so
// building a longer "+" chain does not nest ExprConcat inside ExprConcat.
func flattenConcat(e doc.Expr) []doc.Expr {
	if e.Kind == doc.ExprConcat {
		return e.Parts
	}
	return []doc.Expr{e}
}

func opString(t token.Token) string {
	switch t {
	case token.EQL:
		return "=="
	case token.NEQ:
		return "!="
	case token.LSS:
		return "<"
	case token.LEQ:
		return "<="
	case token.GTR:
		return ">"
	case token.GEQ:
		return ">="
	default:
		return t.String()
	}
}
