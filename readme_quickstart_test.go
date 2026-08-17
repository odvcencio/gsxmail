package gsxmail_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// quickstartSnippetPattern extracts the fenced Go code block between
// README.md's own "gsxmail:quickstart-snippet:start"/"...:end" HTML
// comment markers — the exact "60-second quick start" step 3 snippet a
// reader copy-pastes.
var quickstartSnippetPattern = regexp.MustCompile("(?s)<!-- gsxmail:quickstart-snippet:start -->\\s*```go\\n(.*?)```\\s*<!-- gsxmail:quickstart-snippet:end -->")

// TestReadmeQuickstartCompiles is the launch-gate M7 finding's own proof
// bar: the README's "60-second quick start" snippet must actually
// compile, not just read plausibly. It extracts the snippet verbatim
// (readme_quickstart_test.go never hand-copies it, so the two cannot
// drift), builds a temporary module exactly as steps 1-2 describe —
// `go mod init example.com/myapp`, then a copy of
// examples/quickstart/emails — writes the snippet as that module's
// main.go, and runs `go build` on it.
func TestReadmeQuickstartCompiles(t *testing.T) {
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	readme, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := quickstartSnippetPattern.FindSubmatch(readme)
	if m == nil {
		t.Fatal("README.md has no gsxmail:quickstart-snippet:start/end marker pair around a ```go block")
	}
	snippet := string(m[1])
	if !strings.Contains(snippet, "package main") {
		t.Fatalf("extracted snippet does not look like a Go source file:\n%s", snippet)
	}

	// mod is the temp module step 1 ("go mod init example.com/myapp")
	// describes, built directly (no `go mod init` subprocess needed: its
	// whole effect for this test is one go.mod line plus the require and
	// replace pair every other outside-module test in this suite already
	// uses to resolve every dependency from the local module cache with
	// no network call).
	mod := t.TempDir()
	repoGoMod, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	repoGoSum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(mod, "go.mod"), quickstartGoMod(string(repoGoMod), repoRoot))
	writeFile(t, filepath.Join(mod, "go.sum"), string(repoGoSum))
	writeFile(t, filepath.Join(mod, "main.go"), snippet)

	// Step 2: copy examples/quickstart's own emails/ folder in, unchanged.
	srcEmails := filepath.Join(repoRoot, "examples", "quickstart", "emails")
	entries, err := os.ReadDir(srcEmails)
	if err != nil {
		t.Fatal(err)
	}
	dstEmails := filepath.Join(mod, "emails")
	if err := os.MkdirAll(dstEmails, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcEmails, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dstEmails, e.Name()), string(data))
	}

	build := runGo(t, mod, "build", "-buildvcs=false", "./...")
	if build.err != nil {
		t.Fatalf("README's quickstart snippet does not compile:\n%s\n--- snippet ---\n%s", build.output, snippet)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type goResult struct {
	output string
	err    error
}

// runGo runs `go <args...>` with dir as its working directory.
func runGo(t *testing.T, dir string, args ...string) goResult {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return goResult{output: string(out), err: err}
}

// quickstartGoMod builds the temp module's go.mod: "module example.com/
// myapp" (matching the snippet's own import path for its copied emails
// package), repoGoMod's require block reused verbatim (so every
// dependency version resolves from the same already-populated local
// module cache the rest of this suite relies on), plus a require+replace
// pair pointing m31labs.dev/gsxmail at repoRoot on disk.
func quickstartGoMod(repoGoMod, repoRoot string) string {
	lines := strings.SplitN(repoGoMod, "\n", 2)
	rest := ""
	if len(lines) == 2 {
		rest = lines[1]
	}
	var b strings.Builder
	b.WriteString("module example.com/myapp\n")
	b.WriteString(rest)
	b.WriteString("\nrequire m31labs.dev/gsxmail v0.0.0\n\n")
	b.WriteString("replace m31labs.dev/gsxmail => " + filepath.ToSlash(repoRoot) + "\n")
	return b.String()
}
