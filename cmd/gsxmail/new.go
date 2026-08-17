package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// runNew implements `gsxmail new`: it scaffolds one starter template so a
// new project has a working *.gsx file, a matching props struct, and a
// sample fixture to render or check immediately, instead of an empty
// --dir. It writes three files and touches nothing else; the caller edits
// them freely afterward.
func runNew(args []string) error {
	if len(args) == 0 || len(args[0]) > 0 && args[0][0] == '-' {
		return fmt.Errorf("new requires the template name as its first argument\n\n%s", usageText())
	}
	rawName := args[0]

	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	dir := fs.String("dir", "emails", "directory to write the new template into")
	pkg := fs.String("package", "emails", "generated Go package name")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("new takes one template name and flags only, got extra argument %q\n\n%s", fs.Arg(0), usageText())
	}

	// A bare name ("Welcome") gets the same "Email" suffix `gsxmail
	// import` gives a caller-supplied --name — one convention for the
	// generated component name across both scaffolding verbs.
	templateName := rawName
	if !strings.HasSuffix(templateName, "Email") {
		templateName += "Email"
	}
	stem := lowerFirst(strings.TrimSuffix(templateName, "Email"))
	if stem == "" {
		return fmt.Errorf("new requires a template name with at least one letter before \"Email\", got %q\n\n%s", rawName, usageText())
	}
	propsName := templateName + "Props"

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return err
	}

	gsxPath := filepath.Join(*dir, stem+".gsx")
	goPath := filepath.Join(*dir, stem+".go")
	jsonPath := filepath.Join(*dir, stem+".props.json")

	files := []struct {
		path    string
		content string
	}{
		{gsxPath, newTemplateGSX(*pkg, templateName, propsName)},
		{goPath, newTemplateGo(*pkg, propsName)},
		{jsonPath, newTemplateSampleJSON()},
	}
	for _, f := range files {
		if _, err := os.Stat(f.path); err == nil {
			return fmt.Errorf("new: %s already exists; remove it or choose a different name", f.path)
		}
	}
	for _, f := range files {
		if err := os.WriteFile(f.path, []byte(f.content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", f.path, err)
		}
		fmt.Println(f.path)
	}
	return nil
}

// lowerFirst lower-cases name's first rune only, giving the file-stem
// convention the quickstart example itself follows ("WelcomeEmail" ->
// "welcome"): the rest of a CamelCase name keeps its own casing so a
// multi-word name ("PasswordResetEmail") reads clearly as one filename
// ("passwordReset.gsx") instead of losing its word boundaries.
func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func newTemplateGSX(pkg, templateName, propsName string) string {
	return fmt.Sprintf(`package %s

func %s(props %s) Node {
    return <email.Shell
        wordmark={props.Wordmark}
        shortCode="OK"
        tagline="TAGLINE"
        title={props.Title}
        lang="en"
        preheader={props.Preheader}>
        <email.Headline title={props.Title} lede={props.Lede} />
        <email.CTA label={props.CTALabel} href={props.CTAHref} />
        <email.Footer signoff={props.Signoff} note={props.Note} />
    </email.Shell>
}
`, pkg, templateName, propsName)
}

func newTemplateGo(pkg, propsName string) string {
	return fmt.Sprintf(`// Package %s holds this project's email templates.
package %s

// %s fills the scaffolded template. The caller formats every field
// before rendering; gsxmail only owns markup. Edit these fields, the
// matching *.gsx template, and the sample JSON together.
type %s struct {
	Wordmark  string // shown top-left in the Shell header
	Title     string // the <title> and the Headline's own title
	Preheader string // the inbox preview line; gsxmail truncates past 150 runes
	Lede      string // the Headline's supporting sentence
	CTALabel  string // the button's own label
	CTAHref   string // must be https, http, or mailto
	Signoff   string // the Footer's closing line, e.g. "— The Team"
	Note      string // the Footer's small-print line
}
`, pkg, pkg, propsName, propsName)
}

func newTemplateSampleJSON() string {
	return `{
  "Wordmark": "ACME",
  "Title": "Hello",
  "Preheader": "A short preview line for the inbox.",
  "Lede": "One supporting sentence about what happened.",
  "CTALabel": "TAKE ACTION →",
  "CTAHref": "https://example.com",
  "Signoff": "— The Team",
  "Note": "You are receiving this email because of your account."
}
`
}
