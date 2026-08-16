package typesafe

import "reflect"

// helperSignature reflects on a registered Options.Helpers value and
// reports its argument count and its first return value's dialect Kind.
// ok is false when fn is not a func, or is variadic (v1's helper calls are
// arity-checked exactly; a variadic helper cannot be arity-checked this
// way and is rejected as outside the dialect, EM010).
func helperSignature(fn any) (arity int, ret Kind, ok bool) {
	rt := reflect.TypeOf(fn)
	if rt == nil || rt.Kind() != reflect.Func || rt.IsVariadic() {
		return 0, KindUnknown, false
	}
	ret = KindUnknown
	if rt.NumOut() > 0 {
		ret = classifyReflectKind(rt.Out(0))
	}
	return rt.NumIn(), ret, true
}

func classifyReflectKind(t reflect.Type) Kind {
	switch t.Kind() {
	case reflect.String:
		return KindString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return KindInt
	case reflect.Float32, reflect.Float64:
		return KindFloat
	case reflect.Bool:
		return KindBool
	case reflect.Slice, reflect.Array:
		return KindSlice
	default:
		return KindOther
	}
}
