package cmd

import (
	"errors"
	"io"
	"os"
	"testing"
)

// fakePrompter returns queued passwords and records the prompts it was given.
type fakePrompter struct {
	passwords [][]byte
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

func TestTerminalPrompterRequiresTerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})

	p := terminalPrompter{in: r, out: io.Discard}
	if _, err := p.Password("Password: "); err == nil {
		t.Fatal("Password() on a pipe returned a nil error")
	}
}
