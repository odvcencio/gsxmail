package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"m31labs.dev/gsxmail/internal/lint"
)

// caniemailDataURL is the one network call gsxmail ever makes. Every other
// verb, including check and render, stays offline (spec section 10).
const caniemailDataURL = "https://www.caniemail.com/api/data.json"

// runMatrix implements `gsxmail matrix`.
func runMatrix(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("matrix requires a subcommand (refresh)\n\n%s", usageText())
	}
	switch args[0] {
	case "refresh":
		return runMatrixRefresh(args[1:])
	default:
		return fmt.Errorf("unknown matrix subcommand %q\n\n%s", args[0], usageText())
	}
}

// runMatrixRefresh implements `gsxmail matrix refresh`: it downloads the
// caniemail dataset, trims it with the same rules the embedded snapshot
// was built with, prints the support diffs against the file it is about
// to overwrite, and rewrites that file. It is meant to run from a
// checkout of the gsxmail module itself (--out defaults to
// lint/snapshot.json, the embedded file's real path), producing a change
// a maintainer reviews and commits like any other — this command never
// commits on its own.
func runMatrixRefresh(args []string) error {
	fs := flag.NewFlagSet("matrix refresh", flag.ContinueOnError)
	out := fs.String("out", filepath.Join("lint", "snapshot.json"), "path to the embedded snapshot file to rewrite")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("matrix refresh takes flags only, got extra argument %q\n\n%s", fs.Arg(0), usageText())
	}

	fmt.Fprintln(os.Stderr, "gsxmail: fetching", caniemailDataURL)
	raw, err := fetchCaniemailData()
	if err != nil {
		return fmt.Errorf("fetching %s: %w", caniemailDataURL, err)
	}

	next, encoded, err := lint.BuildSnapshot(raw)
	if err != nil {
		return fmt.Errorf("trimming caniemail dataset: %w", err)
	}

	var prev *lint.Snapshot
	if prevBytes, readErr := os.ReadFile(*out); readErr == nil {
		prev, _ = lint.ParseSnapshot(prevBytes)
	}
	printSupportDiff(prev, next)

	if err := os.WriteFile(*out, encoded, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", *out, err)
	}
	fmt.Printf("gsxmail: wrote %s (snapshot date %s, %d properties, %d clients)\n",
		*out, next.Date, len(next.Properties), len(next.Clients))
	return nil
}

func fetchCaniemailData() ([]byte, error) {
	resp, err := http.Get(caniemailDataURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// printSupportDiff prints every per-client support change, added property,
// and removed property between prev (the file being overwritten, or nil
// the first time this command runs) and next (the freshly trimmed data),
// so a reviewer sees exactly what a `gsxmail matrix refresh` commit
// changes before approving it.
func printSupportDiff(prev, next *lint.Snapshot) {
	if prev == nil {
		fmt.Println("gsxmail: no previous snapshot to diff against; writing the first one")
		return
	}
	if prev.Date != next.Date {
		fmt.Printf("gsxmail: snapshot date %s -> %s\n", prev.Date, next.Date)
	}

	names := make([]string, 0, len(next.Properties))
	for name := range next.Properties {
		names = append(names, name)
	}
	sort.Strings(names)

	changed := 0
	for _, name := range names {
		ps := next.Properties[name]
		old, existed := prev.Properties[name]
		if !existed {
			fmt.Printf("  + %s (new)\n", name)
			changed++
			continue
		}
		for _, c := range next.Clients {
			was, now := old.Support[c.ID], ps.Support[c.ID]
			if was != now {
				fmt.Printf("  ~ %s / %s: %s -> %s\n", name, c.Label, was, now)
				changed++
			}
		}
	}
	removed := make([]string, 0)
	for name := range prev.Properties {
		if _, ok := next.Properties[name]; !ok {
			removed = append(removed, name)
		}
	}
	sort.Strings(removed)
	for _, name := range removed {
		fmt.Printf("  - %s (removed)\n", name)
		changed++
	}

	if changed == 0 {
		fmt.Println("gsxmail: no support changes since the last snapshot")
	}
}
