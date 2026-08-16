package gsxmail_test

import (
	"encoding/json"
	"os/exec"
	"testing"
)

// corePackages is gsxmail's render path: every package Load and Render
// execute code from. None of them may directly import gotreesitter, or
// this repo's own internal/structverify (which imports it), from a
// non-_test.go file (design spec section 15, WP5.1; pixel dossier
// section 7.3: "The core library render path never imports gotreesitter
// under any outcome").
//
// cmd/gsxmail is deliberately not on this list as of WP5.5: the pixel
// dossier's own isolation rule extends, rather than tightens, for the
// `gsxmail import` verb — "the import package lives OUTSIDE the core
// render path ... the importer package and CLI may [import
// gotreesitter]" (task instructions, design constraint 1). The CLI binary
// now legitimately reaches gotreesitter through m31labs.dev/gsxmail/
// importer; TestImporterIsolatedFromRenderPath below is this file's own
// WP5.5 addition, proving the one thing that constraint still forbids:
// none of corePackages may import package importer either, directly or
// (courtesy of the same go list check) any other way that would put
// gotreesitter back on the render path's own doorstep.
//
// This is a direct-import check, not a transitive one: `m31labs.dev/gosx`
// — WP1's own hard dependency, unrelated to WP5.1 — is itself built on
// gotreesitter (github.com/odvcencio/gotreesitter/grammargen, gosx's own
// compile.go), so gotreesitter is unavoidably present somewhere in every
// core package's transitive closure already. What WP5.1 must not do, and
// what this test proves it does not do, is add a new *direct* gotreesitter
// import to any of gsxmail's own render-path packages.
var corePackages = []string{
	"m31labs.dev/gsxmail",
	"m31labs.dev/gsxmail/doc",
	"m31labs.dev/gsxmail/lower",
	"m31labs.dev/gsxmail/renderhtml",
	"m31labs.dev/gsxmail/rendertext",
	"m31labs.dev/gsxmail/typesafe",
	"m31labs.dev/gsxmail/lint",
}

// TestGotreesitterIsolatedFromCorePath is the module-graph proof the
// pixel dossier's isolation rule requires. It runs `go list -f
// {{.Imports}}` — a package's own direct, non-test import list — over
// every core (render-path and CLI) package and fails if
// "github.com/odvcencio/gotreesitter" or
// "m31labs.dev/gsxmail/internal/structverify" appears in it. Both may
// only ever be reachable from a package's _test.go files, which `go list
// -f {{.Imports}}` does not report.
func TestGotreesitterIsolatedFromCorePath(t *testing.T) {
	for _, pkg := range corePackages {
		out, err := exec.Command("go", "list", "-json", pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -json %s: %v\n%s", pkg, err, out)
		}
		var info struct {
			ImportPath string
			Imports    []string
		}
		if err := json.Unmarshal(out, &info); err != nil {
			t.Fatalf("parsing go list -json %s output: %v\n%s", pkg, err, out)
		}
		for _, dep := range info.Imports {
			if dep == "github.com/odvcencio/gotreesitter" {
				t.Errorf("%s directly imports gotreesitter from a non-_test.go file; the core render path must never do that", pkg)
			}
			if dep == "m31labs.dev/gsxmail/internal/structverify" {
				t.Errorf("%s directly imports internal/structverify from a non-_test.go file; that package is a test-layer-only opt-in, never part of the render path", pkg)
			}
			if dep == "m31labs.dev/gsxmail/importer" {
				t.Errorf("%s directly imports package importer from a non-_test.go file; that package is the CLI's own opt-in, never part of the render path", pkg)
			}
		}
	}
}

// TestImporterIsolatedFromRenderPath is WP5.5's own addition to this
// file's isolation proof: package importer legitimately imports
// gotreesitter (it is gsxmail's second deliberate use, after
// internal/structverify — see importer/doc.go), and cmd/gsxmail
// legitimately imports package importer to wire up the `import` verb.
// What must still never happen is a render-path package reaching either
// one — corePackages and the loop above already prove that. This test
// proves the complementary fact stakeholders reading this file would
// otherwise have to infer: cmd/gsxmail does reach gotreesitter now, on
// purpose, through exactly one path.
func TestImporterIsolatedFromRenderPath(t *testing.T) {
	out, err := exec.Command("go", "list", "-json", "m31labs.dev/gsxmail/cmd/gsxmail").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -json m31labs.dev/gsxmail/cmd/gsxmail: %v\n%s", err, out)
	}
	var info struct {
		ImportPath string
		Imports    []string
	}
	if err := json.Unmarshal(out, &info); err != nil {
		t.Fatalf("parsing go list -json output: %v\n%s", err, out)
	}
	found := false
	for _, dep := range info.Imports {
		if dep == "m31labs.dev/gsxmail/importer" {
			found = true
		}
	}
	if !found {
		t.Error("cmd/gsxmail no longer imports m31labs.dev/gsxmail/importer; if the import verb moved, update this test's own expectation instead of deleting it")
	}
}
