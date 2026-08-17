package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"

	"m31labs.dev/gsxmail"
)

// runCheck implements `gsxmail check`: it runs Load and prints every
// finding (spec section 10). It exits 1 the moment any error-severity
// finding exists, whether Load failed closed on it (no Set at all) or a
// warning-only load still carries lower-severity findings. This verb runs
// no network call; Load never does.
//
// Options.Helpers is always empty here: the CLI has no consumer Go
// process to read a real helper map from, so it cannot tell a genuinely
// unregistered helper from one the consumer's own program registers.
// EM014/EM015 findings from this command are the CLI's own limitation,
// not the template's; validate helper bindings with Set.Check() inside
// your own test instead, where Options.Helpers is real (see the README).
func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	dir := fs.String("dir", "emails", "directory of *.gsx templates to check")
	format := fs.String("format", "text", `output format: "text" or "json"`)
	severity := fs.String("severity", "all", `minimum severity to print: "all", "warn", or "error"`)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("check takes flags only, got extra argument %q\n\n%s", fs.Arg(0), usageText())
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("check: --format must be \"text\" or \"json\", got %q", *format)
	}
	if *severity != "all" && *severity != "warn" && *severity != "error" {
		return fmt.Errorf("check: --severity must be \"all\", \"warn\", or \"error\", got %q", *severity)
	}

	set, loadErr := gsxmail.Load(os.DirFS(*dir), gsxmail.Options{Dir: *dir})

	var diags []gsxmail.Diagnostic
	var lintErr *gsxmail.LintError
	switch {
	case loadErr == nil:
		diags = set.Check()
	case errors.As(loadErr, &lintErr):
		diags = lintErr.Diagnostics
	default:
		return fmt.Errorf("loading %s: %w", *dir, loadErr)
	}

	// hasErrorSeverity always sees the full, unfiltered list: --severity
	// only narrows what prints, never the exit code a CI job depends on.
	exitNonZero := hasErrorSeverity(diags)

	diags = sortDiagnostics(filterSeverity(diags, *severity))

	if *format == "json" {
		if err := printDiagnosticsJSON(diags); err != nil {
			return err
		}
	} else {
		printDiagnosticsText(diags, *dir)
	}

	if exitNonZero {
		os.Exit(1)
	}
	return nil
}

// filterSeverity keeps only diags at or above min ("all" keeps
// everything, "warn" keeps warn and error, "error" keeps error only) —
// polish item 9's own `--severity` flag.
func filterSeverity(diags []gsxmail.Diagnostic, min string) []gsxmail.Diagnostic {
	if min == "all" {
		return diags
	}
	out := diags[:0:0]
	for _, d := range diags {
		if min == "error" && d.Severity != "error" {
			continue
		}
		out = append(out, d)
	}
	return out
}

// sortDiagnostics orders diags by file, then line, then column — grouped
// by file, and in source order within each file (polish item 9) — a
// stable sort, so two findings at the same position keep whatever order
// the lint walker itself produced them in.
func sortDiagnostics(diags []gsxmail.Diagnostic) []gsxmail.Diagnostic {
	sort.SliceStable(diags, func(i, j int) bool {
		a, b := diags[i], diags[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
	return diags
}

func hasErrorSeverity(diags []gsxmail.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == "error" {
			return true
		}
	}
	return false
}

func printDiagnosticsJSON(diags []gsxmail.Diagnostic) error {
	if diags == nil {
		diags = []gsxmail.Diagnostic{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(diags)
}

func printDiagnosticsText(diags []gsxmail.Diagnostic, dir string) {
	if len(diags) == 0 {
		fmt.Printf("gsxmail check: %s: no findings\n", dir)
		return
	}
	errs, warns := 0, 0
	for _, d := range diags {
		fmt.Println(d.String())
		if d.Severity == "error" {
			errs++
		} else {
			warns++
		}
	}
	fmt.Printf("gsxmail check: %s: %d error(s), %d warning(s)\n", dir, errs, warns)
}
