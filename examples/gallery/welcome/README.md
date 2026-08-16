# Welcome

An onboarding email. It confirms a new account and points the recipient
to their first three steps.

## Components

`Shell`, `Headline`, `PickList`, `Button` (variant `"primary"`), `Footer`.

## Files

- `welcome.gsx` — the template.
- `welcome.go` — `WelcomeProps`, the Go struct the template reads.
- `welcome.props.json` — fixture props.
- `welcome.html` / `welcome.txt` — the golden render, `gsxmail.DefaultTheme()`.

## Render it

```go
set, err := gsxmail.Load(os.DirFS("welcome"), gsxmail.Options{})
parts, err := set.Render("WelcomeEmail", WelcomeProps{
    Product:  "Acme",
    Name:     "Ada",
    LoginURL: "https://acme.example/login",
})
```
