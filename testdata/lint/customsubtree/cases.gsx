// Package customsubtree is a regression fixture: WP2's per-property style
// lint (EM101/EM102) must still fire on a raw element nested inside
// <email.Shell> — the Custom pass-through position WP3 adds — exactly as
// it already does on a bare top-level element.
package customsubtree

func CustomBadStyle() Node {
    return <email.Shell wordmark="X" shortCode="X" tagline="X" title="X" lang="en">
        <table style="border-radius: 4px;">
            <tr><td>cell</td></tr>
        </table>
    </email.Shell>
}
