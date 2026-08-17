package gsxmail

import "errors"

// Sentinel errors: every error Load and Render can return wraps one of
// these, so a caller can classify a failure with errors.Is instead of
// matching an error message's own text (which this package makes no
// promise to keep stable). Each sentinel's own doc comment names exactly
// which call sites wrap it.
var (
	// ErrCompile wraps a gosx compile failure: a *.gsx file under fsys
	// does not parse as valid gosx source at all. Load returns this
	// before ever running the email lint.
	ErrCompile = errors.New("gsxmail: compile error")

	// ErrLower wraps a lower.Lower failure: a template cleared the email
	// lint, but Lower still rejects it (an unsupported root, or a
	// construct — such as <If>/<Each> — the lint recognizes as valid
	// dialect but this release cannot yet render). Load returns this
	// after every template in fsys has already cleared the lint.
	ErrLower = errors.New("gsxmail: lower error")

	// ErrDuplicateTemplate wraps Load's own "template already declared in
	// another file" failure: two components across the loaded *.gsx
	// files share one name.
	ErrDuplicateTemplate = errors.New("gsxmail: duplicate template name")

	// ErrUnknownTemplate wraps Render's own "no template named %q"
	// failure: name does not match any template Load found in fsys.
	ErrUnknownTemplate = errors.New("gsxmail: unknown template")

	// ErrPropsMismatch wraps Render's own props-type-mismatch failure:
	// props is a struct (or a pointer to one) whose own type name differs
	// from the template's declared props type. A map[string]any props
	// value is exempt — it has no named Go type to compare in the first
	// place (the render CLI's own path, since it decodes JSON with no
	// static Go type to target).
	ErrPropsMismatch = errors.New("gsxmail: props type mismatch")

	// ErrNilProps wraps Render's own nil-pointer-props failure: props is
	// a nil pointer to the template's declared props type.
	ErrNilProps = errors.New("gsxmail: nil props")

	// ErrResolve wraps every other doc.Resolve failure at render time:
	// props is neither a struct nor a map[string]any, a field the
	// template reads is unset, an interpolated value is not a string,
	// integer, float, or bool, or an internal resolution error. Most of
	// these are also provable at Load time by the email lint (EM012,
	// EM013) for a named Go props type; ErrResolve is the render-time
	// backstop for the cases Load-time checking cannot reach — a
	// map[string]any props value (no named Go type to check) chief among
	// them. See "Two layers, one guarantee" in the README.
	ErrResolve = errors.New("gsxmail: resolve error")
)
