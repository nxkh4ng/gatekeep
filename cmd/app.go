package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// app holds the dependencies shared by all commands.
type app struct {
	vault    string
	prompter Prompter
}

// errNotImplemented marks commands that exist in the tree but are filled in
// by later milestones.
var errNotImplemented = errors.New("not implemented yet")

func notImplemented(*cobra.Command, []string) error {
	return errNotImplemented
}
