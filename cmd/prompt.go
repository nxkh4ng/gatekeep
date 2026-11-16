package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Prompter reads secrets from the user.
type Prompter interface {
	Password(prompt string) ([]byte, error)
}

// terminalPrompter reads passwords from a terminal without echoing them.
type terminalPrompter struct {
	in  *os.File
	out io.Writer
}

func newTerminalPrompter() Prompter {
	return terminalPrompter{in: os.Stdin, out: os.Stderr}
}

func (p terminalPrompter) Password(prompt string) ([]byte, error) {
	if !term.IsTerminal(int(p.in.Fd())) {
		return nil, errors.New("cannot read password: stdin is not a terminal")
	}
	if _, err := fmt.Fprint(p.out, prompt); err != nil {
		return nil, fmt.Errorf("writing prompt: %w", err)
	}
	password, err := term.ReadPassword(int(p.in.Fd()))
	_, _ = fmt.Fprintln(p.out)
	if err != nil {
		return nil, fmt.Errorf("reading password: %w", err)
	}
	return password, nil
}

// requiredPassword re-prompts until the user enters a non-empty password.
func requiredPassword(cmd *cobra.Command, p Prompter, prompt string) ([]byte, error) {
	for {
		password, err := p.Password(prompt)
		if err != nil {
			return nil, err
		}
		if len(password) > 0 {
			return password, nil
		}
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Password cannot be empty."); err != nil {
			return nil, err
		}
	}
}
