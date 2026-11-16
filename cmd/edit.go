package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nxkh4ng/gatekeep/core"
	"github.com/spf13/cobra"
)

func newEditCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <query>",
		Short: "Edit an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEdit(cmd, a, args[0])
		},
	}
	cmd.Flags().String("title", "", "new title")
	cmd.Flags().String("username", "", "new username")
	cmd.Flags().String("url", "", "new URL")
	cmd.Flags().String("notes", "", "new notes")
	cmd.Flags().Bool("password", false, "prompt for a new password")
	return cmd
}

func runEdit(cmd *cobra.Command, a *app, query string) error {
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
	current, err := resolveEntry(cmd, a.prompter, entries, query)
	if err != nil {
		return err
	}

	updated, err := editedEntry(cmd, a, current)
	if err != nil {
		return err
	}
	if updated == current {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No changes.")
		return err
	}

	if err := v.Update(current.ID, updated); err != nil {
		return fmt.Errorf("updating entry: %w", err)
	}
	if err := v.SaveFile(a.vault); err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated %s\n", updated.Title)
	return err
}

// editedEntry applies the provided field flags, or runs the wizard when no
// field flag was given.
func editedEntry(cmd *cobra.Command, a *app, current core.Entry) (core.Entry, error) {
	flags := cmd.Flags()
	passwordFlag, err := flags.GetBool("password")
	if err != nil {
		return core.Entry{}, err
	}

	if !flags.Changed("title") && !flags.Changed("username") && !flags.Changed("url") &&
		!flags.Changed("notes") && !flags.Changed("password") {
		return editWizard(cmd, a.prompter, current)
	}

	updated := current
	if flags.Changed("title") {
		if updated.Title, err = flags.GetString("title"); err != nil {
			return core.Entry{}, err
		}
		if updated.Title = strings.TrimSpace(updated.Title); updated.Title == "" {
			return core.Entry{}, errors.New("title is required")
		}
	}
	if flags.Changed("username") {
		if updated.Username, err = flags.GetString("username"); err != nil {
			return core.Entry{}, err
		}
		if updated.Username = strings.TrimSpace(updated.Username); updated.Username == "" {
			return core.Entry{}, errors.New("username is required")
		}
	}
	if flags.Changed("url") {
		if updated.URL, err = flags.GetString("url"); err != nil {
			return core.Entry{}, err
		}
		updated.URL = strings.TrimSpace(updated.URL)
	}
	if flags.Changed("notes") {
		if updated.Notes, err = flags.GetString("notes"); err != nil {
			return core.Entry{}, err
		}
		updated.Notes = strings.TrimSpace(updated.Notes)
	}
	if flags.Changed("password") && passwordFlag {
		password, err := a.prompter.Password("New password (Enter to keep): ")
		if err != nil {
			return core.Entry{}, err
		}
		if len(password) > 0 {
			updated.Password = string(password)
		}
		clear(password)
	}
	return updated, nil
}

// editWizard prompts for every field, keeping the current value on Enter.
func editWizard(cmd *cobra.Command, p Prompter, current core.Entry) (core.Entry, error) {
	updated := current
	var err error
	if updated.Title, err = keepRequiredLine(cmd, p, "Title", current.Title); err != nil {
		return core.Entry{}, err
	}
	if updated.Username, err = keepRequiredLine(cmd, p, "Username", current.Username); err != nil {
		return core.Entry{}, err
	}

	password, err := p.Password("Password (Enter to keep): ")
	if err != nil {
		return core.Entry{}, err
	}
	if len(password) > 0 {
		updated.Password = string(password)
	}
	clear(password)

	if updated.URL, err = keepLine(p, "URL", current.URL); err != nil {
		return core.Entry{}, err
	}
	if updated.Notes, err = keepLine(p, "Notes", current.Notes); err != nil {
		return core.Entry{}, err
	}
	return updated, nil
}

// keepLine prompts with the current value and keeps it on an empty answer.
func keepLine(p Prompter, label, current string) (string, error) {
	prompt := label + ": "
	if current != "" {
		prompt = fmt.Sprintf("%s [%s]: ", label, current)
	}
	line, err := p.Line(prompt)
	if err != nil {
		return "", err
	}
	if line = strings.TrimSpace(line); line != "" {
		return line, nil
	}
	return current, nil
}

func keepRequiredLine(cmd *cobra.Command, p Prompter, label, current string) (string, error) {
	for {
		value, err := keepLine(p, label, current)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s cannot be empty.\n", label); err != nil {
			return "", err
		}
	}
}
