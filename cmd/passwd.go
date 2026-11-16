package cmd

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func newPasswdCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "passwd",
		Short: "Change the master password",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPasswd(cmd, a)
		},
	}
}

func runPasswd(cmd *cobra.Command, a *app) error {
	v, err := loadVault(cmd, a)
	if err != nil {
		return err
	}
	defer v.Lock()

	password, err := requiredPassword(cmd, a.prompter, "New master password: ")
	if err != nil {
		return err
	}
	defer clear(password)

	confirm, err := requiredPassword(cmd, a.prompter, "Repeat new master password: ")
	if err != nil {
		return err
	}
	defer clear(confirm)

	if !bytes.Equal(password, confirm) {
		return errors.New("master passwords do not match")
	}

	if err := v.ChangePassword(password); err != nil {
		return fmt.Errorf("changing master password: %w", err)
	}
	if err := v.SaveFile(a.vault); err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Master password changed.")
	return err
}
