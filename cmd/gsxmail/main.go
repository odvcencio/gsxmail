// Command gsxmail is the gsxmail CLI. This release ships render, check,
// and matrix refresh; dev (the live preview server) lands in a later
// work package (spec section 10, section 15).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"m31labs.dev/gsxmail"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gsxmail:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	switch args[0] {
	case "render":
		return runRender(args[1:])
	case "check":
		return runCheck(args[1:])
	case "import":
		return runImport(args[1:])
	case "matrix":
		return runMatrix(args[1:])
	case "-h", "--help", "help":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown verb %q\n\n%s", args[0], usageText())
	}
}

func usageError() error {
	return fmt.Errorf("%s", usageText())
}

func usageText() string {
	return `usage: gsxmail <verb> [flags]

verbs:
  render <Template> --dir emails --props file.json [--out dir | --html - | --text -]
      Renders one template with props decoded from JSON. Writes
      <Template>.html and <Template>.txt under --out (default: the
      current directory), or streams one part to stdout.

  check [--dir emails] [--format text|json] [--severity all|warn|error]
      Runs Load and prints every lint finding (spec section 8), sorted by
      file, then line, then column, so findings in the same file group
      together. Exits 1 if any finding is error-severity, even when
      --severity hides it from the printed list. --format json feeds CI
      annotations. --severity narrows the printed list only ("warn"
      prints warn and error; "error" prints error only; "all" is the
      default). This verb never calls the network; it cannot see a
      consumer's registered helpers, so EM014/EM015 findings here mean
      only "the CLI cannot verify this," not "this helper is missing" —
      validate helper bindings with Set.Check() in your own test instead.

  import <in.html> --out <dir> [--name Welcome] [--package emails]
      Reverse-maps an existing email HTML file (MJML compiled output,
      react-email output, or hand-written table soup) onto gsxmail's
      shipped email.* components: gotreesitter's error-tolerant HTML
      grammar parses even malformed markup to a best-effort tree, and
      the mapper never fails closed on a node it cannot place — it
      preserves it inside an email.Custom block instead. Writes
      template.gsx, props.go, theme.go, props.sample.json, and
      IMPORT-REPORT.md under --out (default: the current directory), and
      prints the report's own summary. props.go imports nothing beyond
      the standard library; theme.go is the one generated file that
      imports gsxmail itself, and is safe to delete. This is the one
      gsxmail verb that imports gotreesitter; see the README's "Import
      from existing HTML" section.

  matrix refresh [--out lint/snapshot.json]
      Downloads the caniemail dataset, rewrites the embedded snapshot
      with its date, and prints the per-client support diffs for review.
      This is the one gsxmail command that calls the network. Run it
      from a gsxmail module checkout and commit the result like any
      other reviewed change.

  dev
      Landing in a later gsxmail release.
`
}

func printUsage(w *os.File) {
	fmt.Fprint(w, usageText())
}

func runRender(args []string) error {
	if len(args) == 0 || len(args[0]) > 0 && args[0][0] == '-' {
		return fmt.Errorf("render requires the template name as its first argument\n\n%s", usageText())
	}
	name := args[0]

	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	dir := fs.String("dir", "emails", "directory of *.gsx templates to load")
	propsPath := fs.String("props", "", "path to a JSON file decoded into the template's props")
	out := fs.String("out", ".", "directory to write <Template>.html and <Template>.txt into")
	htmlOut := fs.String("html", "", "stream only the HTML part to this path (\"-\" for stdout)")
	textOut := fs.String("text", "", "stream only the text part to this path (\"-\" for stdout)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("render takes one template name and flags only, got extra argument %q\n\n%s", fs.Arg(0), usageText())
	}

	props, err := loadProps(*propsPath)
	if err != nil {
		return err
	}

	set, err := gsxmail.Load(os.DirFS(*dir), gsxmail.Options{Dir: *dir})
	if err != nil {
		return fmt.Errorf("loading %s: %w", *dir, err)
	}

	parts, err := set.Render(name, props)
	if err != nil {
		return fmt.Errorf("rendering %s: %w", name, err)
	}
	// A render-time diagnostic (today, only EM121) has no source position
	// to format the way gsxmail check's Load-time findings do — it is a
	// property of this one render's output, not of template source — so
	// it prints as "gsxmail: CODE: message" instead of Diagnostic.String's
	// "file:line:col: CODE: message" shape.
	for _, d := range parts.Diagnostics {
		fmt.Fprintf(os.Stderr, "gsxmail: %s: %s\n", d.Code, d.Message)
	}

	if *htmlOut != "" || *textOut != "" {
		if *htmlOut != "" {
			if err := writeOutput(*htmlOut, parts.HTML); err != nil {
				return err
			}
		}
		if *textOut != "" {
			if err := writeOutput(*textOut, parts.Text); err != nil {
				return err
			}
		}
		return nil
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	htmlPath := filepath.Join(*out, name+".html")
	textPath := filepath.Join(*out, name+".txt")
	if err := os.WriteFile(htmlPath, []byte(parts.HTML), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(textPath, []byte(parts.Text), 0o644); err != nil {
		return err
	}
	fmt.Println(htmlPath)
	fmt.Println(textPath)
	return nil
}

// loadProps decodes path's JSON object into a map[string]any. The CLI has
// no static Go type for a template's declared props struct (that
// resolution is WP2's typesafe/ package via go/types), so it renders
// against the generic map path gsxmail.Set.Render also accepts from
// library callers that pass a struct.
func loadProps(path string) (map[string]any, error) {
	if path == "" {
		return map[string]any{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var props map[string]any
	if err := json.Unmarshal(data, &props); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	return props, nil
}

func writeOutput(path, content string) error {
	if path == "-" {
		_, err := fmt.Print(content)
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
