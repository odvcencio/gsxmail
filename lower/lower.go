// Package lower converts a compiled gosx ir.Program into a gsxmail
// doc.EmailDoc: it resolves email.* stdlib tags, inlines their attribute
// expressions into doc.Expr values, and rejects anything outside the WP1
// component set (Shell, Signal, Headline, Panel/PanelRow, CTA,
// PickList/Item, Footer). This is pipeline stage 2 (link) and stage 3
// (lower to EmailDoc) from the design spec, section 7.2.
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
	blocks, err := lowerBlocks(prog, root.Children, propsName)
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
	if s.Wordmark, err = attrExpr(root.Attrs, "wordmark", propsName); err != nil {
		return s, err
	}
	if s.ShortCode, err = attrExpr(root.Attrs, "shortCode", propsName); err != nil {
		return s, err
	}
	if s.Tagline, err = attrExpr(root.Attrs, "tagline", propsName); err != nil {
		return s, err
	}
	if s.Title, err = attrExpr(root.Attrs, "title", propsName); err != nil {
		return s, err
	}
	if s.Lang, err = attrExpr(root.Attrs, "lang", propsName); err != nil {
		return s, err
	}
	return s, nil
}

// lowerBlocks walks children (a Shell's or a container block's children in
// source order), skipping whitespace-only text nodes the gosx parser
// inserts between JSX-style element siblings, and lowers every email.*
// component child to a doc.Block.
func lowerBlocks(prog *ir.Program, children []ir.NodeID, propsName string) ([]doc.Block, error) {
	var blocks []doc.Block
	for _, id := range children {
		n := prog.NodeAt(id)
		if isBlankText(n) {
			continue
		}
		if n.Kind != ir.NodeComponent {
			return nil, fmt.Errorf("unexpected %s inside <email.Shell>; only email.* components are allowed there", kindName(n.Kind))
		}
		ns, local := splitTag(n.Tag)
		if ns != "email" {
			return nil, fmt.Errorf("component <%s> is not an email.* component and is not declared in this template set", n.Tag)
		}
		block, err := lowerBlock(prog, local, n, propsName)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func lowerBlock(prog *ir.Program, local string, n *ir.Node, propsName string) (doc.Block, error) {
	switch local {
	case "Signal":
		text, err := attrExpr(n.Attrs, "text", propsName)
		if err != nil {
			return nil, err
		}
		return doc.Signal{Text: text}, nil
	case "Headline":
		title, err := attrExpr(n.Attrs, "title", propsName)
		if err != nil {
			return nil, err
		}
		lede, err := attrExpr(n.Attrs, "lede", propsName)
		if err != nil {
			return nil, err
		}
		return doc.Headline{Title: title, Lede: lede}, nil
	case "Panel":
		rows, err := lowerPanelRows(prog, n.Children, propsName)
		if err != nil {
			return nil, err
		}
		return doc.Panel{Rows: rows}, nil
	case "CTA":
		label, err := attrExpr(n.Attrs, "label", propsName)
		if err != nil {
			return nil, err
		}
		href, err := attrExpr(n.Attrs, "href", propsName)
		if err != nil {
			return nil, err
		}
		return doc.CTA{Label: label, Href: href}, nil
	case "PickList":
		title, err := attrExpr(n.Attrs, "title", propsName)
		if err != nil {
			return nil, err
		}
		items, err := lowerItems(prog, n.Children, propsName)
		if err != nil {
			return nil, err
		}
		return doc.PickList{Title: title, Items: items}, nil
	case "Footer":
		signoff, err := attrExpr(n.Attrs, "signoff", propsName)
		if err != nil {
			return nil, err
		}
		note, err := attrExpr(n.Attrs, "note", propsName)
		if err != nil {
			return nil, err
		}
		return doc.Footer{Signoff: signoff, Note: note}, nil
	default:
		return nil, fmt.Errorf("component <email.%s> is not on the WP1 email.* component list (Shell, Signal, Headline, Panel/PanelRow, CTA, PickList/Item, Footer)", local)
	}
}

func lowerPanelRows(prog *ir.Program, children []ir.NodeID, propsName string) ([]doc.PanelRow, error) {
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
		label, err := attrExpr(n.Attrs, "label", propsName)
		if err != nil {
			return nil, err
		}
		value, err := attrExpr(n.Attrs, "value", propsName)
		if err != nil {
			return nil, err
		}
		rows = append(rows, doc.PanelRow{Label: label, Value: value})
	}
	return rows, nil
}

func lowerItems(prog *ir.Program, children []ir.NodeID, propsName string) ([]doc.Item, error) {
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
		text, err := inlineContent(prog, n.Children, propsName)
		if err != nil {
			return nil, err
		}
		items = append(items, doc.Item{Text: text})
	}
	return items, nil
}

// inlineContent concatenates an element's text and expression children, in
// source order, into one Expr. Whitespace-only text nodes are dropped.
func inlineContent(prog *ir.Program, children []ir.NodeID, propsName string) (doc.Expr, error) {
	var parts []doc.ExprPart
	for _, id := range children {
		n := prog.NodeAt(id)
		switch n.Kind {
		case ir.NodeText:
			if strings.TrimSpace(n.Text) == "" {
				continue
			}
			parts = append(parts, doc.ExprPart{Literal: n.Text})
		case ir.NodeExpr:
			e, err := parseExprSource(n.Text, propsName)
			if err != nil {
				return doc.Expr{}, err
			}
			parts = append(parts, e.Parts...)
		default:
			return doc.Expr{}, fmt.Errorf("<email.Item> content must be text or {expression}, got %s", kindName(n.Kind))
		}
	}
	return doc.Expr{Parts: parts}, nil
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

// attrExpr finds the named attribute on attrs and compiles its value to an
// Expr. A missing attribute lowers to an empty static Expr rather than an
// error: WP1 has no lint stage (that is WP2's typesafe/ package), so a
// missing required attribute surfaces as an empty rendered field instead of
// a check-time diagnostic.
func attrExpr(attrs []ir.Attr, name, propsName string) (doc.Expr, error) {
	for _, a := range attrs {
		if a.Name != name {
			continue
		}
		switch a.Kind {
		case ir.AttrStatic:
			return doc.Static(a.Value), nil
		case ir.AttrExpr:
			return parseExprSource(a.Expr, propsName)
		default:
			return doc.Expr{}, fmt.Errorf("attribute %s: only static and {expression} values are supported in v1", name)
		}
	}
	return doc.Expr{}, nil
}

// parseExprSource compiles a raw Go expression source string (as recorded
// on ir.Attr.Expr / ir.Node.Text for NodeExpr) into a doc.Expr. The email
// dialect's expression grammar in v1 (spec section 6.1) is a narrow subset:
// string literals, direct props field reads, and string concatenation with
// +. Anything else is a lower-time error — WP2's typesafe/ package replaces
// this with a full go/types-checked evaluator and the EM010/EM011/EM012
// catalog; this function enforces the same fail-closed default without that
// machinery.
func parseExprSource(src, propsName string) (doc.Expr, error) {
	expr, err := parser.ParseExpr(src)
	if err != nil {
		return doc.Expr{}, fmt.Errorf("expression %q does not parse as a Go expression: %w", src, err)
	}
	var parts []doc.ExprPart
	if err := collectExprParts(expr, propsName, &parts); err != nil {
		return doc.Expr{}, fmt.Errorf("expression %q: %w", src, err)
	}
	return doc.Expr{Parts: parts}, nil
}

func collectExprParts(n ast.Expr, propsName string, parts *[]doc.ExprPart) error {
	switch e := n.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return fmt.Errorf("literal %s is not a string; the email dialect allows only string literals in v1", e.Value)
		}
		s, err := strconv.Unquote(e.Value)
		if err != nil {
			return fmt.Errorf("cannot unquote string literal %s: %w", e.Value, err)
		}
		*parts = append(*parts, doc.ExprPart{Literal: s})
		return nil
	case *ast.SelectorExpr:
		ident, ok := e.X.(*ast.Ident)
		if !ok || ident.Name != propsName {
			return fmt.Errorf("is not in the email dialect; allowed: %s.<field>, string literals, and string + ", propsName)
		}
		*parts = append(*parts, doc.ExprPart{Field: e.Sel.Name})
		return nil
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return fmt.Errorf("operator %q is not in the email dialect; only string + is allowed in v1", e.Op)
		}
		if err := collectExprParts(e.X, propsName, parts); err != nil {
			return err
		}
		return collectExprParts(e.Y, propsName, parts)
	case *ast.ParenExpr:
		return collectExprParts(e.X, propsName, parts)
	default:
		return fmt.Errorf("is not in the email dialect; allowed: props field paths, string literals, and string +")
	}
}
