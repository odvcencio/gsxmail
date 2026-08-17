package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
)

// TestRunNewGeneratesLoadableTemplate is polish item 10's own proof: the
// scaffolded template, props struct, and sample fixture Load, Check clean,
// and Render together, exactly as the quickstart's own trio does.
func TestRunNewGeneratesLoadableTemplate(t *testing.T) {
	dir := t.TempDir()
	if err := runNew([]string{"Welcome", "--dir", dir, "--package", "emails"}); err != nil {
		t.Fatalf("runNew: %v", err)
	}

	for _, name := range []string{"welcome.gsx", "welcome.go", "welcome.props.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}

	set, err := gsxmail.Load(os.DirFS(dir), gsxmail.Options{Dir: dir})
	if err != nil {
		t.Fatalf("Load(scaffolded dir): %v", err)
	}
	if diags := set.Check(); len(diags) != 0 {
		t.Errorf("Check() on scaffolded template = %v, want none", diags)
	}

	data, err := os.ReadFile(filepath.Join(dir, "welcome.props.json"))
	if err != nil {
		t.Fatalf("reading sample props: %v", err)
	}
	var props map[string]any
	if err := json.Unmarshal(data, &props); err != nil {
		t.Fatalf("decoding sample props: %v", err)
	}
	parts, err := set.Render("WelcomeEmail", props)
	if err != nil {
		t.Fatalf("Render(WelcomeEmail, sample props): %v", err)
	}
	if parts.HTML == "" || parts.Text == "" {
		t.Error("Render produced an empty HTML or text part")
	}
}

// TestRunNewNameSuffixConvention confirms `new` follows the same "Email"
// suffix rule as `import`: a bare name gains the suffix, a name that
// already carries it is left alone, and the file stem lower-cases only
// the name's first rune.
func TestRunNewNameSuffixConvention(t *testing.T) {
	cases := []struct {
		name     string
		wantGSX  string
		wantFunc string
	}{
		{"Welcome", "welcome.gsx", "WelcomeEmail"},
		{"WelcomeEmail", "welcome.gsx", "WelcomeEmail"},
		{"PasswordReset", "passwordReset.gsx", "PasswordResetEmail"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := runNew([]string{tc.name, "--dir", dir}); err != nil {
				t.Fatalf("runNew(%q): %v", tc.name, err)
			}
			data, err := os.ReadFile(filepath.Join(dir, tc.wantGSX))
			if err != nil {
				t.Fatalf("reading %s: %v", tc.wantGSX, err)
			}
			if !strings.Contains(string(data), "func "+tc.wantFunc+"(") {
				t.Errorf("%s does not declare func %s(...); got:\n%s", tc.wantGSX, tc.wantFunc, data)
			}
		})
	}
}

// TestRunNewRefusesToOverwrite protects a caller's own edits: `new` never
// clobbers a file that is already there.
func TestRunNewRefusesToOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := runNew([]string{"Welcome", "--dir", dir}); err != nil {
		t.Fatalf("first runNew: %v", err)
	}
	if err := runNew([]string{"Welcome", "--dir", dir}); err == nil {
		t.Error("second runNew over the same name succeeded, want a refuse-to-overwrite error")
	}
}
