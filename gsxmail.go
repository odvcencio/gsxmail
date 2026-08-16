// Package gsxmail compiles gosx email templates to a deterministic
// multipart pair: pixel-targeted HTML plus a 72-column plain-text twin,
// from one source tree. See the package README for the full pitch and
// guarantees.
package gsxmail

import (
	"fmt"
	"io/fs"
	"reflect"
	"sort"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gsxmail/doc"
	"m31labs.dev/gsxmail/lower"
	"m31labs.dev/gsxmail/renderhtml"
	"m31labs.dev/gsxmail/rendertext"
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

	// Helpers registers pure functions callable from templates. WP1's
	// expression grammar has no call syntax, so this field is accepted
	// but unused until a later work package adds helper calls.
	Helpers map[string]any

	// MaxHTMLBytes is the Gmail-clip size budget: 0 selects the default
	// 100,000 bytes; -1 disables the check. WP1 stores this value but
	// does not enforce it; the EM120/EM121 budget check is a later work
	// package (spec section 8, section 15 WP3).
	MaxHTMLBytes int
}

// Diagnostic is one check-time finding. WP1's Load performs no linting, so
// Check always returns nil; the type exists so WP2's EM catalog can fill it
// in without changing the Set API.
type Diagnostic struct {
	File     string
	Line     int
	Col      int
	Code     string
	Severity string
	Message  string
}

// String formats d as "file:line:col: CODE: message" (spec section 8).
func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d:%d: %s: %s", d.File, d.Line, d.Col, d.Code, d.Message)
}

// Set is an immutable, goroutine-safe collection of compiled templates.
type Set struct {
	templates map[string]*compiledTemplate
	names     []string
	opts      Options
}

type compiledTemplate struct {
	doc       *doc.EmailDoc
	propsType string
}

// Load compiles every *.gsx file under fsys and lowers each declared
// component to an EmailDoc. WP1 performs no separate lint pass (spec
// section 8's EM catalog is a later work package); a component that gosx
// cannot compile, or that lower.Lower rejects (an unsupported root, an
// unknown email.* tag, or an expression outside the v1 grammar), fails
// Load closed: the whole call returns an error and no Set.
func Load(fsys fs.FS, opts Options) (*Set, error) {
	if (opts.Theme == Theme{}) {
		opts.Theme = DefaultTheme()
	}
	if opts.MaxHTMLBytes == 0 {
		opts.MaxHTMLBytes = 100_000
	}

	templates := make(map[string]*compiledTemplate)
	walkErr := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".gsx") {
			return nil
		}
		src, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("gsxmail: reading %s: %w", path, err)
		}
		prog, err := gosx.Compile(src)
		if err != nil {
			return fmt.Errorf("gsxmail: compiling %s: %w", path, err)
		}
		for _, c := range prog.Components {
			emailDoc, err := lower.Lower(prog, c.Name)
			if err != nil {
				return fmt.Errorf("gsxmail: %s: %w", path, err)
			}
			if _, dup := templates[c.Name]; dup {
				return fmt.Errorf("gsxmail: %s: template %q is already declared in another file", path, c.Name)
			}
			templates[c.Name] = &compiledTemplate{doc: emailDoc, propsType: c.PropsType}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)

	return &Set{templates: templates, names: names, opts: opts}, nil
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

// Check runs the email lint without rendering. WP1 has no lint stage, so
// Check always returns nil; a later work package fills in the EM catalog.
func (s *Set) Check() []Diagnostic {
	return nil
}

// checkPropsType rejects a struct props value whose named type does not
// match the template's declared props type. A map[string]any (the render
// CLI's path, ahead of WP2's go/types-resolved decoding) has no named Go
// type to compare and skips this check.
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
