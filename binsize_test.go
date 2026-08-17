package gsxmail_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// cliSizeBudgetBytes is m6's own CI size gate (launch-gate findings, CI
// sketch, M9): the tagged, HTML-only CLI build must stay under 30 MB.
// The build measures roughly 24.4 MB in practice (see README's "Import
// from existing HTML" section for the exact figure); the 30 MB ceiling
// leaves headroom for gotreesitter's own growth without gsxmail's CI
// going red on a release that changed nothing gsxmail owns.
const cliSizeBudgetBytes = 30 * 1024 * 1024

// TestCLIBinarySizeUnderBudget builds `gsxmail` with the recommended
// default tags (`grammar_subset grammar_subset_html`, m6: README's "The
// CLI" section leads with this exact command now) and asserts the
// resulting binary stays under cliSizeBudgetBytes. It skips under
// -short: it invokes a real `go build`, which is slow enough to be worth
// opting out of for a quick local test loop.
func TestCLIBinarySizeUnderBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI binary; skipped under -short")
	}
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "gsxmail")
	cmd := exec.Command("go", "build",
		"-tags", "grammar_subset grammar_subset_html",
		"-o", bin, "./cmd/gsxmail")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the tagged CLI: %v\n%s", err, out)
	}
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("tagged CLI binary: %d bytes (%.1f MB)", info.Size(), float64(info.Size())/(1024*1024))
	if info.Size() > cliSizeBudgetBytes {
		t.Errorf("tagged CLI binary is %d bytes; the m6 size gate is %d bytes (30 MB)", info.Size(), int64(cliSizeBudgetBytes))
	}
}
