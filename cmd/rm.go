package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newRmCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <query>",
		Short: "Delete an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(cmd, a, args[0])
		},
	}
}

func runRm(cmd *cobra.Command, a *app, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return errors.New("query is required")
	}

	v, err := loadVault(cmd, a)
	if err != nil {
		return err
	}
	defer v.Lock()

	entries, err := v.Entries()
	if err != nil {
		return err
	}
	entry, err := resolveEntry(cmd, a.prompter, entries, query)
	if err != nil {
		return err
	}

	answer, err := a.prompter.Line(fmt.Sprintf("Delete %q (%s)? [y/N] ", entry.Title, entry.Username))
	if err != nil {
		return err
	}
	if answer = strings.ToLower(strings.TrimSpace(answer)); answer != "y" && answer != "yes" {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
		return err
	}

	if err := v.Delete(entry.ID); err != nil {
		return fmt.Errorf("deleting entry: %w", err)
	}
	if err := v.SaveFile(a.vault); err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s\n", entry.Title)
	return err
}
