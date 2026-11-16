package cmd

import (
	"bufio"
	"errors"
	"io"
	"os"
	"testing"
)

// fakePrompter returns queued passwords and lines, recording every prompt.
type fakePrompter struct {
	passwords [][]byte
	lines     []string
	prompts   []string
	err       error
}

func (f *fakePrompter) Password(prompt string) ([]byte, error) {
	f.prompts = append(f.prompts, prompt)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.passwords) == 0 {
		return nil, errors.New("fakePrompter: no passwords left")
	}
	password := f.passwords[0]
	f.passwords = f.passwords[1:]
	return password, nil
}

func (f *fakePrompter) Line(prompt string) (string, error) {
	f.prompts = append(f.prompts, prompt)
	if f.err != nil {
		return "", f.err
	}
	if len(f.lines) == 0 {
		return "", errors.New("fakePrompter: no lines left")
	}
	line := f.lines[0]
	f.lines = f.lines[1:]
	return line, nil
}

func TestTerminalPrompterRequiresTerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})

	p := &terminalPrompter{in: r, out: io.Discard, r: bufio.NewReader(r)}
	if _, err := p.Password("Password: "); err == nil {
		t.Fatal("Password() on a pipe returned a nil error")
	}
	if _, err := p.Line("Title: "); err == nil {
		t.Fatal("Line() on a pipe returned a nil error")
	}
}
