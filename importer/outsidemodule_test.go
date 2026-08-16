package importer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
)

// TestCheckSucceedsOutsideModule reproduces the launch-gate B3 finding's
// own failing case (probes-gsxmail/res/inmod): `gsxmail check` on the
// importer's own generated output, run with the process's current working
// directory nowhere near this module or the consumer module the generated
// files live in. Before B3's fix this produced a wall of misleading EM012
// "no such field" findings (props resolution silently gave up and fell
// through to the generic props-is-nil path); it must now either resolve
// cleanly or report the real cause as EM192, never EM012.
//
// The reproduction builds a real, separate consumer module on disk (its
// own go.mod, requiring m31labs.dev/gsxmail via a local replace, copying
// this module's own go.mod/go.sum requires so every dependency resolves
// from the local module cache with no network call), writes one corpus
// fixture's generated template.gsx/props.go/theme.go into it, then loads
// that consumer module's emails directory through gsxmail.Load with
// Options.Dir set (exactly as `gsxmail check` does) while the test
// process's own working directory is a third, unrelated temp directory —
// t.Chdir proves the CWD independence in-process, without needing to
// spawn and rely on the `go` toolchain shelling out from a subprocess.
func TestCheckSucceedsOutsideModule(t *testing.T) {
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	repoGoMod, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatalf("reading this module's own go.mod: %v", err)
	}
	repoGoSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatalf("reading this module's own go.sum: %v", err)
	}

	src, err := os.ReadFile(filepath.Join("testdata", "corpus", "legacy.html"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(src, "legacy.html", Options{TemplateName: "Newsletter"})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// consumerDir is a module of its own, entirely outside repoRoot: its
	// go.mod requires and replaces m31labs.dev/gsxmail, and reuses
	// repoRoot's own require block verbatim (same versions, same go.sum)
	// so every dependency resolves from the already-populated local
	// module cache — no network call, exactly like every other test in
	// this suite.
	consumerDir := t.TempDir()
	emailsDir := filepath.Join(consumerDir, "emails")
	if err := os.MkdirAll(emailsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	consumerGoMod := buildConsumerGoMod(string(repoGoMod), repoRoot)
	if err := os.WriteFile(filepath.Join(consumerDir, "go.mod"), []byte(consumerGoMod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(consumerDir, "go.sum"), repoGoSum, 0o644); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(emailsDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("template.gsx", res.TemplateGSX)
	write("props.go", res.PropsGo)
	write("theme.go", res.ThemeGo)

	// elsewhere is a third directory, a sibling of consumerDir, with no
	// go.mod of its own anywhere above it up to the filesystem root that
	// gsxmail or consumerDir would be reachable from — the process's
	// current working directory while Load runs, reproducing the
	// examiner's own probe exactly: CWD is not the module that owns the
	// props file being checked.
	elsewhere := t.TempDir()
	t.Chdir(elsewhere)

	set, err := gsxmail.Load(os.DirFS(emailsDir), gsxmail.Options{Dir: emailsDir})
	if err != nil {
		var lintErr *gsxmail.LintError
		if !errors.As(err, &lintErr) {
			t.Fatalf("Load: %v", err)
		}
		for _, d := range lintErr.Diagnostics {
			if d.Code == "EM012" {
				t.Errorf("Load fell through to a misleading EM012 instead of resolving props.go outside the module: %s", d.String())
			}
		}
		t.Fatalf("Load failed outside the module:\n%s", diagsString(lintErr.Diagnostics))
	}

	for _, d := range set.Check() {
		if d.Severity == "error" {
			t.Errorf("Check reported an error-severity finding outside the module: %s", d.String())
		}
	}

	if !strings.Contains(res.PropsGo, "package emails") {
		t.Fatalf("sanity: generated props.go looks wrong:\n%s", res.PropsGo)
	}
}

func diagsString(diags []gsxmail.Diagnostic) string {
	var b strings.Builder
	for _, d := range diags {
		b.WriteString("  " + d.String() + "\n")
	}
	return b.String()
}

// buildConsumerGoMod rewrites repoGoMod (this module's own go.mod text)
// into a new module's go.mod that requires and replaces
// m31labs.dev/gsxmail: same require block (so the same dependency
// versions resolve from the same local module cache, needing no network
// call), a new module path, and an added require+replace pair for
// m31labs.dev/gsxmail itself, pointed at repoRoot on disk.
func buildConsumerGoMod(repoGoMod, repoRoot string) string {
	lines := strings.SplitN(repoGoMod, "\n", 2)
	rest := ""
	if len(lines) == 2 {
		rest = lines[1]
	}
	var b strings.Builder
	b.WriteString("module consumer.local/outsidecheck\n")
	b.WriteString(rest)
	b.WriteString("\nrequire m31labs.dev/gsxmail v0.0.0\n\n")
	b.WriteString("replace m31labs.dev/gsxmail => " + filepath.ToSlash(repoRoot) + "\n")
	return b.String()
}
