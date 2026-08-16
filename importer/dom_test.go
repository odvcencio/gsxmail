package importer

import (
	"os"
	"testing"
)

// TestParseHTMLDomify is a direct check on the domify layer (dom.go)
// against a real gallery golden, ahead of and independent from the
// higher-level mapper tests in selfroundtrip_test.go and
// foreigncorpus_test.go.
func TestParseHTMLDomify(t *testing.T) {
	src, err := os.ReadFile("../examples/gallery/receipt/receipt.html")
	if err != nil {
		t.Fatal(err)
	}
	root, err := parseHTML(src)
	if err != nil {
		t.Fatal(err)
	}
	tables := findAll(root, "table")
	if len(tables) == 0 {
		t.Fatal("expected at least one table")
	}
	title := findFirst(root, "title")
	if title == nil {
		t.Fatal("expected a title element")
	}
	if got := title.innerText(); got != "ACME receipt" {
		t.Errorf("title = %q, want %q", got, "ACME receipt")
	}
	badge := false
	for _, sp := range findAll(root, "span") {
		if sp.innerText() == "PAID" {
			badge = true
		}
	}
	if !badge {
		t.Error("expected to find the PAID badge span")
	}
}
