package gsxmail_test

import (
	"strings"
	"testing"

	"m31labs.dev/gsxmail"
)

// TestNumericAttrsRejectInjection reproduces probes-gsxmail/inject's exact
// payload shapes (B1, launch-gate findings): a template that binds
// email.Spacer's height, email.Hero's width, email.Column's imgWidth, and
// email.Button's link-variant width to props used to write an unescaped
// event handler and a remote url() straight into the rendered attribute or
// style, with zero diagnostics. Render must now reject every one of these
// values instead of writing them.
func TestNumericAttrsRejectInjection(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*NewBlocksProps)
		wantErr string
	}{
		{
			name:    "SpacerHeight",
			mutate:  func(p *NewBlocksProps) { p.SpacerHeight = `20" onmouseover="x" data-evil="` },
			wantErr: `email.Spacer height must be a positive decimal integer`,
		},
		{
			name:    "HeroWidth",
			mutate:  func(p *NewBlocksProps) { p.HeroWidth = `600px; background:url(http://evil.example/track.gif)` },
			wantErr: `email.Hero width must be a positive decimal integer`,
		},
		{
			name:    "ColumnImgWidth",
			mutate:  func(p *NewBlocksProps) { p.Col1ImgWidth = `120" onerror="alert(1)` },
			wantErr: `email.Column imgWidth must be a positive decimal integer`,
		},
		{
			name:    "ButtonLinkWidth",
			mutate:  func(p *NewBlocksProps) { p.LinkWidth = `1px" onclick="alert(1)` },
			wantErr: `email.Button width must be a positive decimal integer`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set := loadNewBlocksSet(t, gsxmail.Options{})
			props := newBlocksFixtureProps()
			tc.mutate(&props)
			parts, err := set.Render("NewBlocks", props)
			if err == nil {
				t.Fatalf("Render succeeded with an injection payload; HTML:\n%s", parts.HTML)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Render error = %q, want a message containing %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestNumericAttrsAcceptPositiveIntegers is the positive-path companion to
// TestNumericAttrsRejectInjection: an ordinary decimal pixel count for each
// of the same four fields renders without error and lands in the HTML
// unescaped-injection-free, exactly as it always has.
func TestNumericAttrsAcceptPositiveIntegers(t *testing.T) {
	set := loadNewBlocksSet(t, gsxmail.Options{})
	parts, err := set.Render("NewBlocks", newBlocksFixtureProps())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, want := range []string{`height="24"`, `width="598"`, `width="120"`, `width:220px`} {
		if !strings.Contains(parts.HTML, want) {
			t.Errorf("HTML part is missing %q", want)
		}
	}
}
