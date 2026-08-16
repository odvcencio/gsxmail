package doc

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Resolve evaluates every Expr in d against props and returns the plain-
// string tree the writers consume. props must be a struct or a
// map[string]any (or a pointer to a struct); anything else is an error.
// Resolve does no I/O and reads no ambient state, so the same EmailDoc and
// the same props always resolve to the same Resolved value.
func (d *EmailDoc) Resolve(props any) (*Resolved, error) {
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

	shell, err := resolveShell(d.Shell, val)
	if err != nil {
		return nil, err
	}
	blocks := make([]ResolvedBlock, 0, len(d.Blocks))
	for _, block := range d.Blocks {
		rb, err := resolveBlock(block, val)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, rb)
	}
	return &Resolved{Shell: shell, Blocks: blocks}, nil
}

func resolveShell(s Shell, props reflect.Value) (ResolvedShell, error) {
	var out ResolvedShell
	var err error
	if out.Wordmark, err = resolveExpr(s.Wordmark, props); err != nil {
		return out, err
	}
	if out.ShortCode, err = resolveExpr(s.ShortCode, props); err != nil {
		return out, err
	}
	if out.Tagline, err = resolveExpr(s.Tagline, props); err != nil {
		return out, err
	}
	if out.Title, err = resolveExpr(s.Title, props); err != nil {
		return out, err
	}
	if out.Lang, err = resolveExpr(s.Lang, props); err != nil {
		return out, err
	}
	return out, nil
}

func resolveBlock(block Block, props reflect.Value) (ResolvedBlock, error) {
	switch b := block.(type) {
	case Signal:
		text, err := resolveExpr(b.Text, props)
		if err != nil {
			return nil, err
		}
		return ResolvedSignal{Text: text}, nil
	case Headline:
		title, err := resolveExpr(b.Title, props)
		if err != nil {
			return nil, err
		}
		lede, err := resolveExpr(b.Lede, props)
		if err != nil {
			return nil, err
		}
		return ResolvedHeadline{Title: title, Lede: lede}, nil
	case Panel:
		rows := make([]ResolvedPanelRow, 0, len(b.Rows))
		for _, row := range b.Rows {
			label, err := resolveExpr(row.Label, props)
			if err != nil {
				return nil, err
			}
			value, err := resolveExpr(row.Value, props)
			if err != nil {
				return nil, err
			}
			rows = append(rows, ResolvedPanelRow{Label: label, Value: value})
		}
		return ResolvedPanel{Rows: rows}, nil
	case CTA:
		label, err := resolveExpr(b.Label, props)
		if err != nil {
			return nil, err
		}
		href, err := resolveExpr(b.Href, props)
		if err != nil {
			return nil, err
		}
		return ResolvedCTA{Label: label, Href: href}, nil
	case PickList:
		title, err := resolveExpr(b.Title, props)
		if err != nil {
			return nil, err
		}
		items := make([]string, 0, len(b.Items))
		for _, item := range b.Items {
			text, err := resolveExpr(item.Text, props)
			if err != nil {
				return nil, err
			}
			items = append(items, text)
		}
		return ResolvedPickList{Title: title, Items: items}, nil
	case Footer:
		signoff, err := resolveExpr(b.Signoff, props)
		if err != nil {
			return nil, err
		}
		note, err := resolveExpr(b.Note, props)
		if err != nil {
			return nil, err
		}
		return ResolvedFooter{Signoff: signoff, Note: note}, nil
	default:
		return nil, fmt.Errorf("gsxmail: internal: unresolved block type %T", block)
	}
}

// resolveExpr evaluates e against props and scrubs the result with
// textSafe, so a control character (or an embedded newline) in a
// recipient-influenced field cannot forge extra lines in the text part or
// smuggle raw control bytes into the HTML part (ported from gridiron's
// internal/emailkit text-safety rule).
func resolveExpr(e Expr, props reflect.Value) (string, error) {
	var b strings.Builder
	for _, part := range e.Parts {
		if part.Field == "" {
			b.WriteString(part.Literal)
			continue
		}
		fv, ok := fieldValue(props, part.Field)
		if !ok {
			return "", fmt.Errorf("gsxmail: props has no field %q", part.Field)
		}
		s, err := formatValue(fv)
		if err != nil {
			return "", fmt.Errorf("gsxmail: props.%s: %w", part.Field, err)
		}
		b.WriteString(s)
	}
	return textSafe(b.String()), nil
}

// fieldValue reads name from props, which is either a struct (field lookup
// by name, the library-caller path) or a map[string]any (key lookup, the
// gsxmail render CLI path: JSON decodes to a map because the CLI has no
// static Go type for the template's declared props struct until WP2's
// typesafe/ package resolves one via go/types).
func fieldValue(props reflect.Value, name string) (reflect.Value, bool) {
	switch props.Kind() {
	case reflect.Struct:
		fv := props.FieldByName(name)
		return fv, fv.IsValid()
	case reflect.Map:
		mv := props.MapIndex(reflect.ValueOf(name))
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
// WP1 enforces it at render time rather than at Load-time lint).
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
