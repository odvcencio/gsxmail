# Alert

A system notification: a signal line, a severity badge, a body note, and
one action button. Severity carries as text and structure — the Signal
line, the Badge's own border and label, and the Note's border-left mark —
never as color alone.

## Components

`Shell`, `Signal`, `Badge` (tone `"critical"` or `"warning"`, chosen with
`<If cond={props.IsCritical}>` since `Badge`'s `tone` attribute is a
static, compile-time literal), `Note`, `Button` (variant `"primary"`).

## Theme

`gsxmail.TerminalTheme()` — dark, mono-forward, `DarkMode: "locked"`.
Alert is the gallery's dark-native demonstration: every color in its
golden is Terminal's own token, with no adaptive style layer to swap —
the theme itself is already the dark presentation.

## Files

- `alert.gsx` — the template.
- `alert.go` — `AlertProps`.
- `alert.props.json` — fixture props.
- `alert.html` / `alert.txt` — the golden render, `gsxmail.TerminalTheme()`.

## Render it

```go
set, err := gsxmail.Load(os.DirFS("alert"), gsxmail.Options{Theme: gsxmail.TerminalTheme()})
parts, err := set.Render("AlertEmail", AlertProps{
    Product:     "Acme",
    Severity:    "CRITICAL",
    IsCritical:  true,
    Message:     "Database replication lag exceeded 5 minutes at 03:14 UTC.",
    ActionLabel: "VIEW INCIDENT →",
    ActionURL:   "https://status.acme.example/incidents/482",
})
```
