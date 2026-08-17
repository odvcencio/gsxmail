package lint

import "testing"

// TestBuildSnapshot exercises the trimming logic `gsxmail matrix refresh`
// runs on freshly downloaded data, against a small synthetic payload shaped
// like the real https://www.caniemail.com/api/data.json — never the
// network itself.
func TestBuildSnapshot(t *testing.T) {
	raw := []byte(`{
		"last_update_date": "2026-08-10 15:16:24 +0000",
		"data": [
			{
				"slug": "css-border-radius",
				"category": "css",
				"stats": {
					"outlook": {"windows": {"2019": "n #1"}},
					"gmail": {"desktop-webmail": {"2020-04": "y", "2022-02": "y #1"}}
				}
			},
			{
				"slug": "css-at-media",
				"category": "css",
				"stats": {}
			},
			{
				"slug": "html-canvas",
				"category": "html",
				"stats": {}
			}
		]
	}`)

	snap, out, err := BuildSnapshot(raw)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if snap.Date != "2026-08-10" {
		t.Errorf("Date = %q, want %q", snap.Date, "2026-08-10")
	}
	if _, ok := snap.Properties["at-media"]; ok {
		t.Error("BuildSnapshot kept an at-rule feature (css-at-media); EM101/EM102 lint style properties only")
	}
	if _, ok := snap.Properties["canvas"]; ok {
		t.Error("BuildSnapshot kept a non-CSS-category feature (html-canvas)")
	}
	br, ok := snap.Properties["border-radius"]
	if !ok {
		t.Fatal(`BuildSnapshot dropped "border-radius"`)
	}
	if got := br.Support["outlook-windows"]; got != "n" {
		t.Errorf(`border-radius outlook-windows support = %q, want "n" (footnote markers must be stripped)`, got)
	}
	if got := br.Support["gmail-web"]; got != "y" {
		t.Errorf(`border-radius gmail-web support = %q, want "y" (the latest version, 2022-02, must win over 2020-04)`, got)
	}
	if got := br.Support["apple-mail-macos"]; got != "u" {
		t.Errorf(`border-radius apple-mail-macos support = %q, want "u" (no data for that client)`, got)
	}

	// out must itself parse back through ParseSnapshot: this is exactly
	// what `gsxmail matrix refresh` writes to snapshot.json.
	if _, err := ParseSnapshot(out); err != nil {
		t.Fatalf("BuildSnapshot's own output does not parse: %v", err)
	}
}

func TestVersionLess(t *testing.T) {
	cases := []struct{ a, b string }{
		{"2019", "2022-02"},
		{"16.80", "2019-08"}, // "16" < "2019" numerically, not lexically
		{"9", "10"},
	}
	for _, tc := range cases {
		if !versionLess(tc.a, tc.b) {
			t.Errorf("versionLess(%q, %q) = false, want true", tc.a, tc.b)
		}
		if versionLess(tc.b, tc.a) {
			t.Errorf("versionLess(%q, %q) = true, want false", tc.b, tc.a)
		}
	}
}
