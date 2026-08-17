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
	"m31labs.dev/gsxmail/internal/doc"
	"m31labs.dev/gsxmail/internal/lint"
	"m31labs.dev/gsxmail/internal/lower"
	"m31labs.dev/gsxmail/internal/typesafe"
	"m31labs.dev/gsxmail/renderhtml"
	"m31labs.dev/gsxmail/rendertext"
)

// Parts is one rendered multipart email.
type Parts struct {
	HTML string
	Text string

	// Diagnostics carries any warning Render itself produced for this one
	// call: EM110 (a CTA/Button href failed the allowed-scheme check —
	// the link drops, the label still renders), EM200 (a dynamic
	// preheader over 150 runes was truncated), and EM121 (the HTML part
	// crossed the 90,000-byte warning line but stayed under budget). It
	// is empty on every Render call that has nothing to report. An
	// error-severity finding never lands here: it makes Render return an
	// error instead (see SizeBudgetError), with a zero Parts.
	Diagnostics []Diagnostic
}

// Options configures a Set.
type Options struct {
	// Theme supplies the palette, fonts, and metrics the HTML writer
	// inlines. The zero value is replaced with DefaultTheme().
	Theme Theme

	// Helpers registers pure functions callable from templates. Load's
	// lint pass validates every helper call against this map: an
	// unregistered name is EM014, and a registered helper called with the
	// wrong number of arguments is EM015. Render invokes the same map by
	// reflection for every ExprCall hole a template's Load already proved
	// registered and arity-checked.
	Helpers map[string]any

	// MaxHTMLBytes is the Gmail-clip size budget (EM120/EM121): 0 selects
	// the default 100,000 bytes; -1 disables both the error and the
	// warning check. Any other negative value fails Load closed with
	// EM201 instead of making every subsequent Render call fail with a
	// confusing "budget: -5 bytes" EM120. Render enforces the budget on
	// every call's rendered HTML part: over budget is a returned
	// *SizeBudgetError with no Parts; over the warning line but still
	// within budget is a returned Parts with one EM121 entry in
	// Diagnostics. The warning line is normally the fixed 90,000 bytes,
	// but scales to 90% of MaxHTMLBytes when MaxHTMLBytes itself is set
	// below that — otherwise a budget tighter than 90,000 bytes made
	// EM121 permanently unreachable, since EM120 would always fire
	// first.
	MaxHTMLBytes int

	// Outlook selects the HTML output contract every template in this Set
	// renders with, unless a template's own <email.Shell outlook="..."
	// attribute overrides it. "" and "ghost-tables" (the default) emit
	// the hardened, bulletproof markup: an Outlook ghost table, doubled
	// DPI-fix widths, td-pair Panel rows, an <h1> Headline title,
	// mso-padding-alt on the CTA, a real StatTable data-table contract,
	// and the border-left Note / spacer-technique Divider. "off" emits
	// the original byte stream unchanged — the parity mode a consumer's
	// own byte- or DOM-equivalence test can pin.
	//
	// This field is the Set-wide default/fallback: a template whose
	// Shell sets its own outlook attribute always wins over this field
	// for that one template; a Shell that leaves outlook unset keeps
	// using this field.
	Outlook string

	// Dir is the real, on-disk directory fsys is rooted at, when fsys is
	// backed by one (typically the same dir string a caller passed to
	// os.DirFS(dir) to build fsys). Set it whenever you can: without it, a
	// template's declared props type that imports another package — this
	// module, a third-party dependency, even the standard library — only
	// resolves that import correctly when the process's current working
	// directory happens to make a relative-path lookup land on the right
	// place (see typesafe.NewResolverAt's own doc comment for the full
	// explanation). Leave it empty for an in-memory or embedded fs.FS
	// with no corresponding real directory — Load then falls back to
	// that CWD-relative resolution.
	Dir string
}

// defaultMaxHTMLBytes is Options.MaxHTMLBytes' zero-value default.
// Gmail's clip point is folklore-documented near 102,400 bytes; this
// budget leaves it margin.
const defaultMaxHTMLBytes = 100_000

// warnHTMLBytes is EM121's fixed warning line: a literal byte count, not
// a fraction of MaxHTMLBytes, so a consumer that raises MaxHTMLBytes
// still gets warned at the same absolute size.
const warnHTMLBytes = 90_000

// SizeBudgetError is the error Render returns when the rendered HTML part
// exceeds Options.MaxHTMLBytes (EM120). Unlike a Load-time LintError
// finding, Diagnostic carries no source position: the budget is a
// property of one Render call's resolved output, not of template source.
type SizeBudgetError struct {
	Diagnostic Diagnostic
}

func (e *SizeBudgetError) Error() string {
	return "gsxmail: " + e.Diagnostic.Code + ": " + e.Diagnostic.Message
}

// Diagnostic is one check-time finding. It is a type alias for
// lint.Diagnostic, so gsxmail's public API never requires a caller to
// import package lint directly.
type Diagnostic = lint.Diagnostic

// LintError is the error Load returns when the email lint finds at least
// one error-severity finding in any loaded template: Load
// fails closed, returning no Set. Diagnostics carries every finding
// gathered across every template in the fs.FS, including any warnings
// found alongside the errors — the same list gsxmail check prints.
type LintError struct {
	Diagnostics []Diagnostic
}

// Error lists every error-severity Diagnostic, one per line, in
// "file:line:col: CODE: message" form.
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

// Load compiles every *.gsx file under fsys, runs the email lint over
// the compiled programs, and — only once every template clears the
// lint — lowers each declared component to an EmailDoc. Load fails
// closed at either stage: a component gosx cannot compile is a plain
// compile error; an error-severity lint finding in any template makes
// Load return the full diagnostic list, as a *LintError, and no Set,
// without ever lowering anything. A component that clears the lint but
// that lower.Lower still rejects (an unsupported root, or a
// construct — such as <If>/<Each> — the lint recognizes as valid dialect
// but this release cannot yet render) fails Load with that plain error.
func Load(fsys fs.FS, opts Options) (*Set, error) {
	if (opts.Theme == Theme{}) {
		opts.Theme = DefaultTheme()
	}
	if opts.MaxHTMLBytes == 0 {
		opts.MaxHTMLBytes = defaultMaxHTMLBytes
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
			return fmt.Errorf("%w: compiling %s: %w", ErrCompile, p, err)
		}
		files = append(files, compiledFile{path: p, prog: prog})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	resolver := typesafe.NewResolverAt(fsys, opts.Dir)
	lintOpts := lint.Options{Helpers: opts.Helpers, Theme: opts.Theme}
	var diagnostics []Diagnostic
	for _, cf := range files {
		diagnostics = append(diagnostics, lint.CheckProgram(cf.path, cf.prog, resolver, path.Dir(cf.path), lintOpts)...)
	}
	// EM140-EM144 check opts.Theme itself, once per Load call — not per
	// template, since every template in this Set shares the same Theme.
	diagnostics = append(diagnostics, lint.CheckTheme(opts.Theme)...)
	// EM194 is Load's own gosx version-skew signal, once per Load call —
	// see checkGosxSkew's own doc comment.
	diagnostics = append(diagnostics, checkGosxSkew()...)
	// EM195 rejects an Options.Outlook value outside the closed set
	// EM172 already enforces per-template on a Shell's own outlook
	// attribute — Options.Outlook is the same structural, compile-time
	// choice at the Set level, and deserves the same fail-closed
	// treatment a case typo or a stray value ("OFF", "gost-tables",
	// "true") once silently defaulted to hardened mode.
	if err := checkOutlookOption(opts.Outlook); err != nil {
		diagnostics = append(diagnostics, *err)
	}
	// EM201 rejects a negative MaxHTMLBytes other than exactly -1 (the
	// documented "disable both checks" sentinel): every other negative
	// value used to reach checkHTMLBudget unvalidated, where
	// `len(html) > maxBytes` is true for any rendered output at all, so
	// every single Render call failed with a confusing "budget: -5
	// bytes" EM120 instead of a clear Load-time diagnostic naming the
	// actual mistake.
	if opts.MaxHTMLBytes < -1 {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "EM201", Severity: "error",
			Message: fmt.Sprintf("Options.MaxHTMLBytes must be 0 (default), a positive byte count, or exactly -1 (disable both checks); got %d", opts.MaxHTMLBytes),
		})
	}
	if hasErrorDiagnostic(diagnostics) {
		return nil, &LintError{Diagnostics: diagnostics}
	}

	templates := make(map[string]*compiledTemplate)
	declaredIn := make(map[string]string)
	for _, cf := range files {
		for _, c := range cf.prog.Components {
			emailDoc, err := lower.Lower(cf.prog, c.Name)
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %w", ErrLower, cf.path, err)
			}
			if firstPath, dup := declaredIn[c.Name]; dup {
				where := "in " + firstPath
				if firstPath == cf.path {
					where = "earlier in the same file"
				}
				return nil, fmt.Errorf("%w: %s: template %q is already declared %s", ErrDuplicateTemplate, cf.path, c.Name, where)
			}
			declaredIn[c.Name] = cf.path
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

// checkOutlookOption implements EM195: Options.Outlook must be "",
// "ghost-tables", or "off" — the same closed set a Shell's own outlook
// attribute is held to (EM172). Returns nil for a valid value.
func checkOutlookOption(outlook string) *Diagnostic {
	switch outlook {
	case "", "ghost-tables", "off":
		return nil
	}
	return &Diagnostic{
		Code: "EM195", Severity: "error",
		Message: fmt.Sprintf(`Options.Outlook must be "", "ghost-tables", or "off"; got %q`, outlook),
	}
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
//
// Render also enforces the Gmail-clip size budget on the rendered HTML
// part (Options.MaxHTMLBytes; EM120/EM121): over budget returns a zero
// Parts and a *SizeBudgetError; over the fixed
// 90,000-byte warning line but still within budget returns the rendered
// Parts with one EM121 entry in Parts.Diagnostics.
func (s *Set) Render(name string, props any) (Parts, error) {
	tmpl, ok := s.templates[name]
	if !ok {
		return Parts{}, fmt.Errorf("%w: no template named %q (loaded: %s)", ErrUnknownTemplate, name, strings.Join(s.names, ", "))
	}
	if tmpl.propsType != "" {
		if err := checkPropsType(props, tmpl.propsType); err != nil {
			return Parts{}, err
		}
	}
	resolved, err := tmpl.doc.ResolveWithHelpers(props, s.opts.Helpers)
	if err != nil {
		return Parts{}, fmt.Errorf("%w: %w", ErrResolve, err)
	}
	// The rendered template's own <email.Shell outlook="..."> attribute
	// overrides this Set's Options.Outlook default/fallback when set; an
	// unset Shell attribute ("") keeps using the Set-wide value.
	outlook := s.opts.Outlook
	if resolved.Shell.Outlook != "" {
		outlook = resolved.Shell.Outlook
	}
	html, renderFindings := renderhtml.WriteWithOptions(resolved, s.opts.Theme, renderhtml.WriteOptions{
		Outlook:   outlook,
		Preheader: resolved.Shell.Preheader,
	})
	parts := Parts{
		HTML: html,
		Text: rendertext.Write(resolved),
	}
	// A disallowed href (EM110) is a visible, non-fatal Diagnostic — a
	// mid-send loop must not die for one bad optional link — never a
	// silently dropped one.
	for _, f := range renderFindings {
		parts.Diagnostics = append(parts.Diagnostics, Diagnostic{Code: f.Code, Message: f.Message, Severity: "warn"})
	}
	diag, sizeErr := checkHTMLBudget(name, parts.HTML, s.opts.MaxHTMLBytes)
	if sizeErr != nil {
		return Parts{}, sizeErr
	}
	if diag != nil {
		parts.Diagnostics = append(parts.Diagnostics, *diag)
	}
	return parts, nil
}

// checkHTMLBudget implements EM120 (error, over Options.MaxHTMLBytes) and
// EM121 (warn, over the fixed 90,000-byte line but still within budget).
// maxBytes == -1 disables both checks. name (the template's own name —
// Render has no
// "fixture" file the way `gsxmail check`'s CLI-level EM120/EM121 does)
// fills the messages' "%s" fixture placeholder.
func checkHTMLBudget(name, html string, maxBytes int) (*Diagnostic, error) {
	if maxBytes == -1 {
		return nil, nil
	}
	n := len(html)
	if n > maxBytes {
		return nil, &SizeBudgetError{Diagnostic: Diagnostic{
			Code:     "EM120",
			Severity: "error",
			Message: fmt.Sprintf(
				"rendered HTML is %d bytes with fixture %s; the budget is %d bytes (Gmail clips near 102400)",
				n, name, maxBytes),
		}}
	}
	warnAt := warnLine(maxBytes)
	if n > warnAt {
		return &Diagnostic{
			Code:     "EM121",
			Severity: "warn",
			Message: fmt.Sprintf(
				"rendered HTML is %d bytes with fixture %s; within budget but above the %d-byte warning line",
				n, name, warnAt),
		}, nil
	}
	return nil, nil
}

// warnLine picks EM121's own warning line for a given MaxHTMLBytes. The
// fixed 90,000-byte line only ever fires when there is room below
// maxBytes for it to fire in at all: a caller who lowers MaxHTMLBytes
// below 90,000 makes the fixed line unreachable (checkHTMLBudget's own
// EM120 check runs first and returns before EM121 ever could), leaving
// EM121 permanently dead for that Set. Scaling the line to 90% of
// maxBytes instead keeps a warn-before-error step for a caller who
// deliberately budgets tighter than Gmail's own clip point.
func warnLine(maxBytes int) int {
	if maxBytes < warnHTMLBytes {
		return maxBytes * 9 / 10
	}
	return warnHTMLBytes
}

// Names lists every loaded template name, sorted.
func (s *Set) Names() []string {
	out := make([]string, len(s.names))
	copy(out, s.names)
	return out
}

// Check returns every finding the email lint produced while loading s,
// without rendering anything. A successfully loaded Set carries only
// warning-severity findings: Load already fails closed on every
// error-severity one, so a Set that exists never has an outstanding
// error. Check does not see EM014/EM015 findings the standalone gsxmail
// check CLI could not: both Load and Check see whatever Options.Helpers s
// was loaded with — that split of responsibilities is a CLI limitation,
// not a library one; see the README.
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
			return fmt.Errorf("%w: props is a nil pointer", ErrNilProps)
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	if got := v.Type().Name(); got != "" && got != declared {
		return fmt.Errorf("%w: props type %s does not match template's declared props type %s", ErrPropsMismatch, got, declared)
	}
	return nil
}
