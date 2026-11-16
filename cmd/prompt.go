package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Prompter reads secrets and field values from the user.
type Prompter interface {
	Password(prompt string) ([]byte, error)
	Line(prompt string) (string, error)
}

// terminalPrompter reads from a terminal without echoing passwords.
type terminalPrompter struct {
	in  *os.File
	out io.Writer
	r   *bufio.Reader
}

func newTerminalPrompter() Prompter {
	return &terminalPrompter{in: os.Stdin, out: os.Stderr, r: bufio.NewReader(os.Stdin)}
}

func (p *terminalPrompter) Password(prompt string) ([]byte, error) {
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

func (p *terminalPrompter) Line(prompt string) (string, error) {
	if !term.IsTerminal(int(p.in.Fd())) {
		return "", errors.New("cannot read input: stdin is not a terminal")
	}
	if _, err := fmt.Fprint(p.out, prompt); err != nil {
		return "", fmt.Errorf("writing prompt: %w", err)
	}
	line, err := p.r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
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

// requiredLine re-prompts until the user enters a non-empty line.
func requiredLine(cmd *cobra.Command, p Prompter, prompt, name string) (string, error) {
	for {
		line, err := p.Line(prompt)
		if err != nil {
			return "", err
		}
		if line = strings.TrimSpace(line); line != "" {
			return line, nil
		}
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s cannot be empty.\n", name); err != nil {
			return "", err
		}
	}
}

// optionalLine reads one line and trims surrounding whitespace.
func optionalLine(p Prompter, prompt string) (string, error) {
	line, err := p.Line(prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
