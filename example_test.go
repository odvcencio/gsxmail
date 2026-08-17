package gsxmail_test

import (
	"fmt"
	"os"
	"strings"

	"m31labs.dev/gsxmail"
	"m31labs.dev/gsxmail/examples/quickstart/emails"
)

// ExampleLoad compiles the quickstart example's one template directory
// and lists every template name it found.
func ExampleLoad() {
	set, err := gsxmail.Load(os.DirFS("examples/quickstart/emails"), gsxmail.Options{Dir: "examples/quickstart/emails"})
	if err != nil {
		fmt.Println("load error:", err)
		return
	}
	fmt.Println(set.Names())
	// Output:
	// [WelcomeEmail]
}

// ExampleSet_Render renders the quickstart example's WelcomeEmail
// template with a typed props value and confirms both parts carry the
// recipient's name.
func ExampleSet_Render() {
	set, err := gsxmail.Load(os.DirFS("examples/quickstart/emails"), gsxmail.Options{Dir: "examples/quickstart/emails"})
	if err != nil {
		fmt.Println("load error:", err)
		return
	}
	parts, err := set.Render("WelcomeEmail", emails.WelcomeProps{
		Name:     "Ada",
		Product:  "Acme",
		LoginURL: "https://acme.example/login",
	})
	if err != nil {
		fmt.Println("render error:", err)
		return
	}
	fmt.Println(strings.Contains(parts.HTML, "Ada"))
	fmt.Println(strings.Contains(parts.Text, "Ada"))
	// Output:
	// true
	// true
}

// ExampleSet_Check runs the email lint over the quickstart example's
// template and prints how many findings it reported — modeling the
// practice the README's own quickstart section recommends: run Check in
// your own CI, without loading twice.
func ExampleSet_Check() {
	set, err := gsxmail.Load(os.DirFS("examples/quickstart/emails"), gsxmail.Options{Dir: "examples/quickstart/emails"})
	if err != nil {
		fmt.Println("load error:", err)
		return
	}
	fmt.Println(len(set.Check()))
	// Output:
	// 0
}

// ExampleTerminalTheme prints TerminalTheme's own dark-mode strategy and
// accent color — a dark, mono-forward named theme that needs no
// Theme.Dark palette, since it is dark-native (DarkMode "locked").
func ExampleTerminalTheme() {
	theme := gsxmail.TerminalTheme()
	fmt.Println(theme.DarkMode)
	fmt.Println(theme.ColorAccent)
	// Output:
	// locked
	// #33E68C
}
