// Package gsxmail compiles gosx email templates to a deterministic
// multipart pair: pixel-targeted HTML plus a 72-column plain-text twin,
// from one source tree. See the package README for the full pitch and
// guarantees.
package gsxmail

import (
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"sort"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/ir"
	"m31labs.dev/gsxmail/doc"
	"m31labs.dev/gsxmail/lint"
	"m31labs.dev/gsxmail/lower"
	"m31labs.dev/gsxmail/renderhtml"
	"m31labs.dev/gsxmail/rendertext"
	"m31labs.dev/gsxmail/typesafe"
)

// Parts is one rendered multipart email.
type Parts struct {
	HTML string
	Text string
}

// Options configures a Set.
type Options struct {
	// Theme supplies the palette, fonts, and metrics the HTML writer
	// inlines. The zero value is replaced with DefaultTheme().
	Theme Theme

	// Helpers registers pure functions callable from templates. Load's
	// lint pass validates every helper call against this map: an
	// unregistered name is EM014, and a registered helper called with the
	// wrong number of arguments is EM015. Runtime helper invocation is a
	// later work package; a template that calls a helper cannot render
	// yet even once it passes the lint (spec section 15, WP3).
	Helpers map[string]any

	// MaxHTMLBytes is the Gmail-clip size budget: 0 selects the default
	// 100,000 bytes; -1 disables the check. WP1 stores this value but
	// does not enforce it; the EM120/EM121 budget check is a later work
	// package (spec section 8, section 15 WP3).
	MaxHTMLBytes int
}

// Diagnostic is one check-time finding (design spec section 8). It is a
// type alias for lint.Diagnostic, so gsxmail's public API never requires a
// caller to import package lint directly.
type Diagnostic = lint.Diagnostic

// LintError is the error Load returns when the email lint (spec section 8)
// finds at least one error-severity finding in any loaded template: Load
// fails closed, returning no Set. Diagnostics carries every finding
// gathered across every template in the fs.FS, including any warnings
// found alongside the errors — the same list gsxmail check prints.
type LintError struct {
	Diagnostics []Diagnostic
}

// Error lists every error-severity Diagnostic, one per line, in
// "file:line:col: CODE: message" form (spec section 8).
func (e *LintError) Error() string {
	var b strings.Builder
	b.WriteString("gsxmail: lint failed:\n")
	for _, d := range e.Diagnostics {
		if d.Severity != "error" {
			continue
		}
		b.WriteString(d.String())
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// Set is an immutable, goroutine-safe collection of compiled templates.
type Set struct {
	templates   map[string]*compiledTemplate
	names       []string
	opts        Options
	diagnostics []Diagnostic
}

type compiledTemplate struct {
	doc       *doc.EmailDoc
	propsType string
}

// Load compiles every *.gsx file under fsys, runs the email lint (spec
// section 8) over the compiled programs, and — only once every template
// clears the lint — lowers each declared component to an EmailDoc. Load
// fails closed at either stage: a component gosx cannot compile is a plain
// compile error; an error-severity lint finding in any template makes Load
// return the full diagnostic list, as a *LintError, and no Set, without
// ever lowering anything (spec section 7.1). A component that clears the
// lint but that lower.Lower still rejects (an unsupported root, or a
// construct — such as <If>/<Each> — the lint recognizes as valid dialect
// but this release cannot yet render) fails Load with that plain error.
func Load(fsys fs.FS, opts Options) (*Set, error) {
	if (opts.Theme == Theme{}) {
		opts.Theme = DefaultTheme()
	}
	if opts.MaxHTMLBytes == 0 {
		opts.MaxHTMLBytes = 100_000
	}

	type compiledFile struct {
		path string
		prog *ir.Program
	}
	var files []compiledFile
	walkErr := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".gsx") {
			return nil
		}
		src, err := fs.ReadFile(fsys, p)
		if err != nil {
			return fmt.Errorf("gsxmail: reading %s: %w", p, err)
		}
		prog, err := gosx.Compile(src)
		if err != nil {
			return fmt.Errorf("gsxmail: compiling %s: %w", p, err)
		}
		files = append(files, compiledFile{path: p, prog: prog})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	resolver := typesafe.NewResolver(fsys)
	lintOpts := lint.Options{Helpers: opts.Helpers}
	var diagnostics []Diagnostic
	for _, cf := range files {
		diagnostics = append(diagnostics, lint.CheckProgram(cf.path, cf.prog, resolver, path.Dir(cf.path), lintOpts)...)
	}
	if hasErrorDiagnostic(diagnostics) {
		return nil, &LintError{Diagnostics: diagnostics}
	}

	templates := make(map[string]*compiledTemplate)
	for _, cf := range files {
		for _, c := range cf.prog.Components {
			emailDoc, err := lower.Lower(cf.prog, c.Name)
			if err != nil {
				return nil, fmt.Errorf("gsxmail: %s: %w", cf.path, err)
			}
			if _, dup := templates[c.Name]; dup {
				return nil, fmt.Errorf("gsxmail: %s: template %q is already declared in another file", cf.path, c.Name)
			}
			templates[c.Name] = &compiledTemplate{doc: emailDoc, propsType: c.PropsType}
		}
	}

	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)

	return &Set{templates: templates, names: names, opts: opts, diagnostics: diagnostics}, nil
}

func hasErrorDiagnostic(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == "error" {
			return true
		}
	}
	return false
}

// Render renders one named template. props must be assignable to the
// template's declared props type; a mismatch is an error, never a zero.
// Rendering is pure: no clock, no network, no maps iterated in order. Same
// Set + same props => same bytes.
func (s *Set) Render(name string, props any) (Parts, error) {
	tmpl, ok := s.templates[name]
	if !ok {
		return Parts{}, fmt.Errorf("gsxmail: no template named %q (loaded: %s)", name, strings.Join(s.names, ", "))
	}
	if tmpl.propsType != "" {
		if err := checkPropsType(props, tmpl.propsType); err != nil {
			return Parts{}, err
		}
	}
	resolved, err := tmpl.doc.Resolve(props)
	if err != nil {
		return Parts{}, err
	}
	return Parts{
		HTML: renderhtml.Write(resolved, s.opts.Theme),
		Text: rendertext.Write(resolved),
	}, nil
}

// Names lists every loaded template name, sorted.
func (s *Set) Names() []string {
	out := make([]string, len(s.names))
	copy(out, s.names)
	return out
}

// Check returns every finding the email lint (spec section 8) produced
// while loading s, without rendering anything. A successfully loaded Set
// carries only warning-severity findings: Load already fails closed on
// every error-severity one, so a Set that exists never has an outstanding
// error. Check does not see EM014/EM015 findings the standalone gsxmail
// check CLI could not: both Load and Check see whatever Options.Helpers s
// was loaded with (spec section 8's "split of responsibilities" is a CLI
// limitation, not a library one — see the README).
func (s *Set) Check() []Diagnostic {
	out := make([]Diagnostic, len(s.diagnostics))
	copy(out, s.diagnostics)
	return out
}

// checkPropsType rejects a struct props value whose named type does not
// match the template's declared props type. This is the render-time half
// of gsxmail's two-layer type check: typesafe/ (package lint's EM010-EM015
// rules) resolves field types from source at Load time, but a props value
// only exists at Render time, so checkPropsType — and doc.Resolve's own
// per-field reflection right after it — re-proves the same guarantee
// against the actual value, fail-closed either way. A map[string]any (the
// render CLI's path, since it decodes JSON with no static Go type to
// target) has no named Go type to compare and skips this check.
func checkPropsType(props any, declared string) error {
	v := reflect.ValueOf(props)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return fmt.Errorf("gsxmail: props is a nil pointer")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	if got := v.Type().Name(); got != "" && got != declared {
		return fmt.Errorf("gsxmail: props type %s does not match template's declared props type %s", got, declared)
	}
	return nil
}
