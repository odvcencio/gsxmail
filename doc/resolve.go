package doc

import (
	"fmt"
	"reflect"
	"strconv"
)

// Resolve evaluates every Expr in d against props and returns the plain-
// string tree the writers consume. It is ResolveWithHelpers(props, nil):
// a template that calls no helper needs nothing more. props must be a
// struct or a map[string]any (or a pointer to a struct); anything else is
// an error. Resolve does no I/O and reads no ambient state, so the same
// EmailDoc and the same props always resolve to the same Resolved value.
func (d *EmailDoc) Resolve(props any) (*Resolved, error) {
	return d.ResolveWithHelpers(props, nil)
}

// ResolveWithHelpers is Resolve plus a helpers map: Options.Helpers,
// passed straight from Set.Render, so a template that calls a registered
// helper (design spec section 6.1; section 15, WP3's "wire actual
// invocation through doc.Resolve") actually invokes it here, not just
// passes Load-time's registration/arity check.
func (d *EmailDoc) ResolveWithHelpers(props any, helpers map[string]any) (*Resolved, error) {
	val := reflect.ValueOf(props)
	for val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, fmt.Errorf("gsxmail: props is a nil pointer")
		}
		val = val.Elem()
	}
	switch val.Kind() {
	case reflect.Struct, reflect.Map:
	default:
		return nil, fmt.Errorf("gsxmail: props must be a struct or a map[string]any, got %s", val.Kind())
	}

	sc := &scope{props: val, bindings: map[string]reflect.Value{}, helpers: helpers}

	shell, err := sc.resolveShell(d.Shell)
	if err != nil {
		return nil, err
	}
	var blocks []ResolvedBlock
	if err := sc.resolveBlocksInto(d.Blocks, &blocks); err != nil {
		return nil, err
	}
	return &Resolved{Shell: shell, Blocks: blocks}, nil
}

func (sc *scope) resolveShell(s Shell) (ResolvedShell, error) {
	var out ResolvedShell
	var err error
	if out.Wordmark, err = sc.resolveText(s.Wordmark); err != nil {
		return out, err
	}
	if out.ShortCode, err = sc.resolveText(s.ShortCode); err != nil {
		return out, err
	}
	if out.Tagline, err = sc.resolveText(s.Tagline); err != nil {
		return out, err
	}
	if out.Title, err = sc.resolveText(s.Title); err != nil {
		return out, err
	}
	if out.Lang, err = sc.resolveText(s.Lang); err != nil {
		return out, err
	}
	return out, nil
}

// resolveBlocksInto resolves every block in blocks, in order, appending
// each block's resolved form (or forms — Each and If can each contribute
// zero or several) to *out.
func (sc *scope) resolveBlocksInto(blocks []Block, out *[]ResolvedBlock) error {
	for _, block := range blocks {
		if err := sc.resolveBlockInto(block, out); err != nil {
			return err
		}
	}
	return nil
}

func (sc *scope) resolveBlockInto(block Block, out *[]ResolvedBlock) error {
	switch b := block.(type) {
	case Signal:
		text, err := sc.resolveText(b.Text)
		if err != nil {
			return err
		}
		*out = append(*out, ResolvedSignal{Text: text})
	case Headline:
		title, err := sc.resolveText(b.Title)
		if err != nil {
			return err
		}
		lede, err := sc.resolveText(b.Lede)
		if err != nil {
			return err
		}
		*out = append(*out, ResolvedHeadline{Title: title, Lede: lede})
	case Panel:
		rows := make([]ResolvedPanelRow, 0, len(b.Rows))
		for _, row := range b.Rows {
			label, err := sc.resolveText(row.Label)
			if err != nil {
				return err
			}
			value, err := sc.resolveText(row.Value)
			if err != nil {
				return err
			}
			rows = append(rows, ResolvedPanelRow{Label: label, Value: value})
		}
		*out = append(*out, ResolvedPanel{Rows: rows})
	case CTA:
		label, err := sc.resolveText(b.Label)
		if err != nil {
			return err
		}
		href, err := sc.resolveText(b.Href)
		if err != nil {
			return err
		}
		*out = append(*out, ResolvedCTA{Label: label, Href: href})
	case PickList:
		title, err := sc.resolveText(b.Title)
		if err != nil {
			return err
		}
		items := make([]string, 0, len(b.Items))
		for _, item := range b.Items {
			text, err := sc.resolveText(item.Text)
			if err != nil {
				return err
			}
			items = append(items, text)
		}
		*out = append(*out, ResolvedPickList{Title: title, Items: items})
	case Footer:
		signoff, err := sc.resolveText(b.Signoff)
		if err != nil {
			return err
		}
		note, err := sc.resolveText(b.Note)
		if err != nil {
			return err
		}
		*out = append(*out, ResolvedFooter{Signoff: signoff, Note: note})
	case Note:
		text, err := sc.resolveText(b.Text)
		if err != nil {
			return err
		}
		*out = append(*out, ResolvedNote{Text: text})
	case Divider:
		*out = append(*out, ResolvedDivider{})
	case StatTable:
		rst, err := sc.resolveStatTable(b)
		if err != nil {
			return err
		}
		*out = append(*out, rst)
	case Custom:
		root, err := sc.resolveCustomNode(b.Root)
		if err != nil {
			return err
		}
		*out = append(*out, ResolvedCustom{Root: root})
	case Each:
		// design spec section 6.5 / section 15 WP3: Of must resolve to a
		// slice; an empty slice contributes nothing. Every element gets
		// its own extended scope (withBinding never shares state across
		// iterations), so Body's own field reads and any nested <Each> or
		// <If> see the right element.
		elements, err := sc.resolveFieldPathElements(b.Of)
		if err != nil {
			return err
		}
		for _, elem := range elements {
			child := sc.withBinding(b.As, elem)
			if err := child.resolveBlocksInto(b.Body, out); err != nil {
				return err
			}
		}
	case If:
		cond, err := sc.resolveBool(b.Cond)
		if err != nil {
			return err
		}
		if cond {
			if err := sc.resolveBlocksInto(b.Body, out); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("gsxmail: internal: unresolved block type %T", block)
	}
	return nil
}

// resolveStatTable resolves a StatTable's title, optional header, and
// every row — literal <email.StatRow> children resolved directly, <Each>
// children expanded once per slice element — in source order, then
// derives MarkRow from whichever resolved row (if any) has Mark true
// (design spec section 6.5, "Mark semantics match emailkit's MarkRow":
// 1-based, 0 meaning none). A template that marks more than one row is not
// rejected — that is an authoring choice, not a dialect violation — but
// only the first marked row's index becomes MarkRow, since both writers
// render exactly one marked row.
func (sc *scope) resolveStatTable(st StatTable) (ResolvedStatTable, error) {
	title, err := sc.resolveText(st.Title)
	if err != nil {
		return ResolvedStatTable{}, err
	}
	header, err := sc.resolveFieldPathStrings(st.Header)
	if err != nil {
		return ResolvedStatTable{}, err
	}

	var rows []ResolvedStatRow
	for _, spec := range st.Rows {
		switch {
		case spec.Literal != nil:
			row, err := sc.resolveStatRow(*spec.Literal)
			if err != nil {
				return ResolvedStatTable{}, err
			}
			rows = append(rows, row)
		case spec.Each != nil:
			elements, err := sc.resolveFieldPathElements(spec.Each.Of)
			if err != nil {
				return ResolvedStatTable{}, err
			}
			for _, elem := range elements {
				child := sc.withBinding(spec.Each.As, elem)
				row, err := child.resolveStatRow(spec.Each.Row)
				if err != nil {
					return ResolvedStatTable{}, err
				}
				rows = append(rows, row)
			}
		}
	}

	markRow := 0
	for i, row := range rows {
		if row.Mark {
			markRow = i + 1
			break
		}
	}
	return ResolvedStatTable{Title: title, Header: header, Rows: rows, MarkRow: markRow}, nil
}

func (sc *scope) resolveStatRow(row StatRow) (ResolvedStatRow, error) {
	cells, err := sc.resolveFieldPathStrings(row.Cells)
	if err != nil {
		return ResolvedStatRow{}, err
	}
	mark, err := sc.resolveBool(row.Mark)
	if err != nil {
		return ResolvedStatRow{}, err
	}
	return ResolvedStatRow{Cells: cells, Mark: mark}, nil
}

func (sc *scope) resolveCustomNode(n CustomNode) (ResolvedCustomNode, error) {
	if n.IsText {
		text, err := sc.resolveText(n.Text)
		if err != nil {
			return ResolvedCustomNode{}, err
		}
		return ResolvedCustomNode{IsText: true, Text: text}, nil
	}
	attrs := make([]ResolvedCustomAttr, len(n.Attrs))
	for i, a := range n.Attrs {
		v, err := sc.resolveText(a.Value)
		if err != nil {
			return ResolvedCustomNode{}, err
		}
		attrs[i] = ResolvedCustomAttr{Name: a.Name, Value: v}
	}
	children := make([]ResolvedCustomNode, len(n.Children))
	for i, c := range n.Children {
		rc, err := sc.resolveCustomNode(c)
		if err != nil {
			return ResolvedCustomNode{}, err
		}
		children[i] = rc
	}
	return ResolvedCustomNode{Tag: n.Tag, Attrs: attrs, Children: children}, nil
}

// fieldValue reads name from container, which is either a struct (field
// lookup by name, the library-caller path) or a map[string]any (key
// lookup, the gsxmail render CLI path: JSON decodes to a map because the
// CLI has no static Go type for the template's declared props struct
// until typesafe/ resolves one via go/types). container is also, inside
// an <Each> body, one loop-bound element rather than the root props value
// — the same struct/map shapes apply either way.
func fieldValue(container reflect.Value, name string) (reflect.Value, bool) {
	switch container.Kind() {
	case reflect.Struct:
		fv := container.FieldByName(name)
		return fv, fv.IsValid()
	case reflect.Map:
		mv := container.MapIndex(reflect.ValueOf(name))
		if !mv.IsValid() {
			return reflect.Value{}, false
		}
		return reflect.ValueOf(mv.Interface()), true
	default:
		return reflect.Value{}, false
	}
}

// formatValue renders a scalar field to its string form. Structs, slices,
// and other non-scalar types are rejected: interpolated values must already
// be strings, integers, floats, or bools (spec section 8, EM013's rule;
// enforced here at render time as the reflection-based second layer of the
// guarantee typesafe/ already checked at Load time).
func formatValue(fv reflect.Value) (string, error) {
	for fv.Kind() == reflect.Interface {
		fv = fv.Elem()
	}
	switch fv.Kind() {
	case reflect.String:
		return fv.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(fv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(fv.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(fv.Float(), 'g', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(fv.Bool()), nil
	case reflect.Invalid:
		return "", fmt.Errorf("field is unset")
	default:
		return "", fmt.Errorf("has type %s; interpolated values must be string, integer, float, or bool", fv.Kind())
	}
}
