// Package darkcustom is EM143's fixture (design spec section 15, WP5.2;
// pixel dossier section 5.3, rule 3): a Custom raw element's literal color
// that matches neither Theme's nor Theme.Dark's own tokens, checked only
// under DarkMode "adaptive" (lint_test.go's TestEM143CustomBlockColor
// loads this directory with an adaptive Theme).
package darkcustom

func OffPaletteColor() Node {
    return <email.Shell wordmark="X" shortCode="X" tagline="X" title="X" lang="en" preheader="ok">
        <div style="color: #123456;">off-palette text</div>
    </email.Shell>
}
