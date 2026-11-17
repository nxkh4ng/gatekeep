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

func newPasswdCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "passwd",
		Short: "Change the master password",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPasswd(cmd, a)
		},
	}
	cmd.Flags().BoolP("backup", "b", false, "keep the previous vault backup, still encrypted with the old password")
	return cmd
}

func runPasswd(cmd *cobra.Command, a *app) error {
	keepBackup, err := cmd.Flags().GetBool("backup")
	if err != nil {
		return err
	}

	lock, err := core.LockFile(a.vault)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()

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
	backup := a.vault + ".bak"
	if keepBackup {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"WARNING: %q is still encrypted with the old master password; remove it when no longer needed.\n", backup)
	} else if err := os.Remove(backup); err != nil && !errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: could not remove %s: %v\n", backup, err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Master password changed.")
	return err
}
