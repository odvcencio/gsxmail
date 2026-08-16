// Command quickstart is gsxmail's 60-second walkthrough: load one
// template, render it with typed props, and write both parts to disk. Run
// it with `go run .` from this directory.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"m31labs.dev/gsxmail"
	"m31labs.dev/gsxmail/examples/quickstart/emails"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "quickstart:", err)
		os.Exit(1)
	}
}

func run() error {
	set, err := gsxmail.Load(os.DirFS("emails"), gsxmail.Options{})
	if err != nil {
		return fmt.Errorf("loading emails/: %w", err)
	}

	data, err := os.ReadFile("emails/welcome.props.json")
	if err != nil {
		return err
	}
	var props emails.WelcomeProps
	if err := json.Unmarshal(data, &props); err != nil {
		return fmt.Errorf("decoding welcome.props.json: %w", err)
	}

	parts, err := set.Render("WelcomeEmail", props)
	if err != nil {
		return fmt.Errorf("rendering WelcomeEmail: %w", err)
	}

	if err := os.WriteFile("welcome.html", []byte(parts.HTML), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("welcome.txt", []byte(parts.Text), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote welcome.html and welcome.txt")
	return nil
}
