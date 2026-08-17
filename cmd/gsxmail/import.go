package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"m31labs.dev/gsxmail/importer"
)

// runImport implements `gsxmail import`: it reverse-maps an existing
// email HTML file onto gsxmail's shipped email.* components and writes
// the result —
// this is the one CLI verb, and the one gsxmail package outside
// internal/structverify's own test-layer use, that imports gotreesitter
// (structural_isolation_test.go's own TestImporterIsolatedFromRenderPath
// proves the render path still never does).
func runImport(args []string) error {
	if len(args) == 0 || len(args[0]) > 0 && args[0][0] == '-' {
		return fmt.Errorf("import requires the source HTML file as its first argument\n\n%s", usageText())
	}
	inPath := args[0]

	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	out := fs.String("out", ".", "directory to write template.gsx, props.go, theme.go, props.sample.json, and IMPORT-REPORT.md into")
	name := fs.String("name", "", `template name ("Welcome" or "WelcomeEmail"); defaults to a name derived from the source <title>`)
	pkg := fs.String("package", "emails", "generated Go package name")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("import takes one source file and flags only, got extra argument %q\n\n%s", fs.Arg(0), usageText())
	}

	html, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inPath, err)
	}

	// Import itself appends the "Email" suffix to a bare *name ("Welcome"
	// -> "WelcomeEmail") when the caller did not already include one —
	// the CLI's --name flag and Options.TemplateName are the same knob,
	// so this verb does not duplicate that rule.
	res, err := importer.Import(html, filepath.Base(inPath), importer.Options{
		PackageName:  *pkg,
		TemplateName: *name,
	})
	if err != nil {
		return fmt.Errorf("importing %s: %w", inPath, err)
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"template.gsx":      res.TemplateGSX,
		"props.go":          res.PropsGo,
		"theme.go":          res.ThemeGo,
		"props.sample.json": res.SamplePropsJSON,
		"IMPORT-REPORT.md":  res.Report.WriteMarkdown(),
	}
	names := []string{"template.gsx", "props.go", "theme.go", "props.sample.json", "IMPORT-REPORT.md"}
	for _, name := range names {
		p := filepath.Join(*out, name)
		if err := os.WriteFile(p, []byte(files[name]), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", p, err)
		}
		fmt.Println(p)
	}

	fmt.Println()
	fmt.Print(res.Report.Summary())
	return nil
}
