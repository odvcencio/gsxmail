package doc

import (
	"fmt"
	"reflect"
	"strconv"
)

// scope is the props-plus-active-loop-bindings environment one Resolve
// call evaluates every Expr and FieldPath against. props is fixed for the
// whole call; bindings grows by one entry per <Each> nesting level
// (withBinding never mutates its receiver's map, so sibling iterations of
// the same <Each> never see each other's binding). helpers is
// Options.Helpers, passed straight through from Set.Render, wiring actual
// invocation through doc.Resolve: Load-time typesafe already proved
// every call's registration and arity;
// scope.callHelper still re-resolves the function value from the same map
// Render was given, so a Set rendered with different Helpers (a caller
// bug, not a template bug) still fails closed here rather than panicking.
type scope struct {
	props    reflect.Value
	bindings map[string]reflect.Value
	helpers  map[string]any
}

// withBinding returns a new scope that extends sc with one more loop
// binding, leaving sc itself untouched.
func (sc *scope) withBinding(name string, v reflect.Value) *scope {
	nb := make(map[string]reflect.Value, len(sc.bindings)+1)
	for k, val := range sc.bindings {
		nb[k] = val
	}
	nb[name] = v
	return &scope{props: sc.props, bindings: nb, helpers: sc.helpers}
}

// resolveFieldRaw reads binding.field (or props.field when binding == "",
// or a loop binding's whole bound value when field == "") and returns its
// reflect.Value, unresolved (still possibly a pointer or interface).
func (sc *scope) resolveFieldRaw(binding, field string) (reflect.Value, error) {
	if binding == "" {
		fv, ok := fieldValue(sc.props, field)
		if !ok {
			return reflect.Value{}, fmt.Errorf("gsxmail: props has no field %q", field)
		}
		return fv, nil
	}
	bv, ok := sc.bindings[binding]
	if !ok {
		return reflect.Value{}, fmt.Errorf("gsxmail: internal: no active loop binding %q", binding)
	}
	if field == "" {
		return bv, nil
	}
	fv, ok := fieldValue(derefValue(bv), field)
	if !ok {
		return reflect.Value{}, fmt.Errorf("gsxmail: loop binding %s has no field %q", binding, field)
	}
	return fv, nil
}

// resolveFieldPathElements resolves fp (a bare slice path — <Each of=...>,
// StatTable.Header, StatRow.Cells) to its
// slice or array elements, each still a raw reflect.Value: Each's own
// caller binds them one at a time; resolveFieldPathStrings formats them.
func (sc *scope) resolveFieldPathElements(fp FieldPath) ([]reflect.Value, error) {
	raw, err := sc.resolveFieldRaw(fp.Binding, fp.Field)
	if err != nil {
		return nil, err
	}
	v := derefValue(raw)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]reflect.Value, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = v.Index(i)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("gsxmail: %s is not a slice or array (got %s)", fieldPathDesc(fp), v.Kind())
	}
}

// resolveFieldPathStrings resolves fp's elements and formats each one
// (StatTable.Header, StatRow.Cells: both are always slices of scalars,
// never slices of struct — EM032-shaped Load-time checking never reaches
// this render-time call for anything else).
func (sc *scope) resolveFieldPathStrings(fp FieldPath) ([]string, error) {
	if fp.IsZero() {
		return nil, nil
	}
	elements, err := sc.resolveFieldPathElements(fp)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(elements))
	for i, elem := range elements {
		s, err := formatValue(derefValue(elem))
		if err != nil {
			return nil, fmt.Errorf("gsxmail: %s[%d]: %w", fieldPathDesc(fp), i, err)
		}
		out[i] = textSafe(s)
	}
	return out, nil
}

func fieldPathDesc(fp FieldPath) string {
	if fp.Binding == "" {
		return "props." + fp.Field
	}
	if fp.Field == "" {
		return fp.Binding
	}
	return fp.Binding + "." + fp.Field
}

// derefValue follows a chain of pointers and interfaces down to the
// concrete value underneath (a nil pointer or interface stays a zero
// reflect.Value, which formatValue and the Kind switches below already
// reject with a clear error rather than panicking).
func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// resolveScalarString evaluates e to its string form: a literal, a props
// or binding field (formatted with formatValue), a helper call's return
// value, or a len() measurement. It is Expr's scalar-context entry point:
// every ExprConcat part, and both of an ExprCompare's operands, resolve
// through it. ExprConcat, ExprNot, and ExprCompare are not themselves
// valid here — Load-time typesafe checking never nests them this way, and
// resolveText/resolveCompare are what call this func on their own
// sub-expressions instead.
func (sc *scope) resolveScalarString(e Expr) (string, error) {
	switch e.Kind {
	case ExprLiteralString, ExprLiteralNumber:
		return e.Literal, nil
	case ExprLiteralBool:
		return strconv.FormatBool(e.Bool), nil
	case ExprField:
		fv, err := sc.resolveFieldRaw(e.Binding, e.Field)
		if err != nil {
			return "", err
		}
		s, err := formatValue(derefValue(fv))
		if err != nil {
			return "", fmt.Errorf("gsxmail: %s: %w", fieldPathDesc(FieldPath{Binding: e.Binding, Field: e.Field}), err)
		}
		return s, nil
	case ExprCall:
		rv, err := sc.callHelper(e)
		if err != nil {
			return "", err
		}
		return formatValue(derefValue(rv))
	case ExprLen:
		n, err := sc.resolveLen(*e.X)
		if err != nil {
			return "", err
		}
		return strconv.Itoa(n), nil
	default:
		return "", fmt.Errorf("gsxmail: internal: expression kind %d is not valid in a scalar context", e.Kind)
	}
}

func (sc *scope) resolveLen(e Expr) (int, error) {
	if e.Kind != ExprField {
		return 0, fmt.Errorf("gsxmail: internal: len() operand must be a field read")
	}
	fv, err := sc.resolveFieldRaw(e.Binding, e.Field)
	if err != nil {
		return 0, err
	}
	v := derefValue(fv)
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return v.Len(), nil
	default:
		return 0, fmt.Errorf("gsxmail: len(%s) operand has type %s; expected a string or slice",
			fieldPathDesc(FieldPath{Binding: e.Binding, Field: e.Field}), v.Kind())
	}
}

// resolveText evaluates e for a text-interpolation hole (Signal.Text,
// PanelRow.Value, ...): an ExprConcat joins its Parts' scalar-string forms
// in order; any other Kind resolves as one scalar. Every resolved value —
// concatenated or bare — passes through textSafe (a control-character
// scrub, ported from gridiron's internal/emailkit),
// exactly once, at the end.
func (sc *scope) resolveText(e Expr) (string, error) {
	if e.Kind == ExprConcat {
		var out string
		for _, part := range e.Parts {
			s, err := sc.resolveScalarString(part)
			if err != nil {
				return "", err
			}
			out += s
		}
		return textSafe(out), nil
	}
	s, err := sc.resolveScalarString(e)
	if err != nil {
		return "", err
	}
	return textSafe(s), nil
}

// resolveBool evaluates e for a bool-context hole (<If cond>, StatRow.Mark).
func (sc *scope) resolveBool(e Expr) (bool, error) {
	switch e.Kind {
	case ExprLiteralBool:
		return e.Bool, nil
	case ExprField:
		fv, err := sc.resolveFieldRaw(e.Binding, e.Field)
		if err != nil {
			return false, err
		}
		v := derefValue(fv)
		if v.Kind() != reflect.Bool {
			return false, fmt.Errorf("gsxmail: %s has type %s; expected bool",
				fieldPathDesc(FieldPath{Binding: e.Binding, Field: e.Field}), v.Kind())
		}
		return v.Bool(), nil
	case ExprNot:
		b, err := sc.resolveBool(*e.X)
		return !b, err
	case ExprCompare:
		return sc.resolveCompare(e)
	case ExprCall:
		rv, err := sc.callHelper(e)
		if err != nil {
			return false, err
		}
		v := derefValue(rv)
		if v.Kind() != reflect.Bool {
			return false, fmt.Errorf("gsxmail: helper %s does not return a bool", e.Call)
		}
		return v.Bool(), nil
	default:
		return false, fmt.Errorf("gsxmail: internal: expression kind %d is not valid in a bool context", e.Kind)
	}
}

// resolveCompare evaluates an ExprCompare: "==" and "!=" compare both
// operands' formatted string form (formatValue already gives every scalar
// Kind — string, int, float, bool — a canonical text form, so this covers
// a bool-vs-literal comparison such as props.Sent == false too);
// ordering operators require both operands to parse as float64.
func (sc *scope) resolveCompare(e Expr) (bool, error) {
	ls, err := sc.resolveScalarString(*e.Left)
	if err != nil {
		return false, err
	}
	rs, err := sc.resolveScalarString(*e.Right)
	if err != nil {
		return false, err
	}
	switch e.Op {
	case "==":
		return ls == rs, nil
	case "!=":
		return ls != rs, nil
	case "<", "<=", ">", ">=":
		lf, errL := strconv.ParseFloat(ls, 64)
		rf, errR := strconv.ParseFloat(rs, 64)
		if errL != nil || errR != nil {
			return false, fmt.Errorf("gsxmail: comparison %q requires numeric operands, got %q and %q", e.Op, ls, rs)
		}
		switch e.Op {
		case "<":
			return lf < rf, nil
		case "<=":
			return lf <= rf, nil
		case ">":
			return lf > rf, nil
		default:
			return lf >= rf, nil
		}
	default:
		return false, fmt.Errorf("gsxmail: internal: unknown comparison operator %q", e.Op)
	}
}

// callHelper invokes the Go function registered under e.Call in
// sc.helpers, converting each argument's resolved scalar-string form to
// that parameter's declared type, and returns its first result. Every
// registered helper is a pure func over scalar types only (Options.Helpers
// doc comment; typesafe.helperSignature enforces the same restriction at
// Load time), so a string round trip through formatValue/coerceScalar for
// every argument is a deliberate simplification, not a lossy one: it lets
// one code path serve every registered signature without a type switch
// per Kind.
func (sc *scope) callHelper(e Expr) (reflect.Value, error) {
	fn, ok := sc.helpers[e.Call]
	if !ok {
		return reflect.Value{}, fmt.Errorf("gsxmail: helper %s is not registered in Options.Helpers", e.Call)
	}
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()
	if fnType.Kind() != reflect.Func {
		return reflect.Value{}, fmt.Errorf("gsxmail: helper %s is not a function", e.Call)
	}
	if fnType.NumIn() != len(e.Args) {
		return reflect.Value{}, fmt.Errorf("gsxmail: helper %s takes %d arguments; the template passes %d", e.Call, fnType.NumIn(), len(e.Args))
	}
	args := make([]reflect.Value, len(e.Args))
	for i, a := range e.Args {
		s, err := sc.resolveScalarString(a)
		if err != nil {
			return reflect.Value{}, err
		}
		argVal, err := coerceScalar(s, fnType.In(i))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("gsxmail: helper %s argument %d: %w", e.Call, i+1, err)
		}
		args[i] = argVal
	}
	out := fnVal.Call(args)
	if len(out) == 0 {
		return reflect.Value{}, fmt.Errorf("gsxmail: helper %s returns no value", e.Call)
	}
	return out[0], nil
}

// coerceScalar parses s into a reflect.Value assignable to target: the
// four Kinds the email dialect ever interpolates (string, integer, float,
// bool — EM013's own list).
func coerceScalar(s string, target reflect.Type) (reflect.Value, error) {
	switch target.Kind() {
	case reflect.String:
		return reflect.ValueOf(s).Convert(target), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("value %q is not an integer", s)
		}
		v := reflect.New(target).Elem()
		v.SetInt(n)
		return v, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("value %q is not an unsigned integer", s)
		}
		v := reflect.New(target).Elem()
		v.SetUint(n)
		return v, nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("value %q is not a number", s)
		}
		v := reflect.New(target).Elem()
		v.SetFloat(f)
		return v, nil
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("value %q is not a bool", s)
		}
		return reflect.ValueOf(b), nil
	default:
		return reflect.Value{}, fmt.Errorf("parameter type %s is not one gsxmail helpers can take (string, integer, float, bool only)", target)
	}
}
