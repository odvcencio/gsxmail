# MagicLink

A sign-in-code email. The code is plain text, in a mono Panel row, never
an image — a screen reader announces it, and it stays selectable for a
recipient who copies it by hand.

## Components

`Shell`, `Headline`, `Panel` (the mono OTP row), `Note` (the expiry
wording), `Button` (variant `"primary"`).

## Files

- `magiclink.gsx` — the template.
- `magiclink.go` — `MagicLinkProps`.
- `magiclink.props.json` — fixture props.
- `magiclink.html` / `magiclink.txt` — the golden render,
  `gsxmail.DefaultTheme()`.

## Render it

```go
set, err := gsxmail.Load(os.DirFS("magiclink"), gsxmail.Options{})
parts, err := set.Render("MagicLinkEmail", MagicLinkProps{
    Product:    "Acme",
    Email:      "ada@example.com",
    Code:       "482 913",
    ExpiryNote: "This code expires in 10 minutes.",
    LoginURL:   "https://acme.example/signin/482913",
})
```
