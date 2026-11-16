package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/nxkh4ng/gatekeep/core"
	"github.com/spf13/cobra"
)

func newInitCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a new vault",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, a)
		},
	}
}

func runInit(cmd *cobra.Command, a *app) error {
	if _, err := os.Stat(a.vault); err == nil {
		return fmt.Errorf("vault already exists at %s", a.vault)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("checking vault: %w", err)
	}

	password, err := requiredPassword(cmd, a.prompter, "Master password: ")
	if err != nil {
		return err
	}
	defer clear(password)

	confirm, err := requiredPassword(cmd, a.prompter, "Repeat master password: ")
	if err != nil {
		return err
	}
	defer clear(confirm)
	if !bytes.Equal(password, confirm) {
		return errors.New("master passwords do not match")
	}

	v, err := core.New(password)
	if err != nil {
		return fmt.Errorf("creating vault: %w", err)
	}
	defer v.Lock()

	if err := v.SaveFile(a.vault); err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Vault created at %s\n", a.vault)
	return err
}
