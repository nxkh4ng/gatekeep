package cmd

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/nxkh4ng/gatekeep/core"
	"github.com/spf13/cobra"
)

func newAddCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Add an entry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, a, args)
		},
	}
	cmd.Flags().String("username", "", "account username (prompted when omitted)")
	cmd.Flags().String("url", "", "entry URL (prompted when omitted)")
	cmd.Flags().String("notes", "", "entry notes (prompted when omitted)")
	return cmd
}

func runAdd(cmd *cobra.Command, a *app, args []string) error {
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

	var title string
	if len(args) == 1 {
		title = strings.TrimSpace(args[0])
	}
	if title == "" {
		title, err = requiredLine(cmd, a.prompter, "Title: ", "Title")
		if err != nil {
			return err
		}
	}

	entries, err := v.Entries()
	if err != nil {
		return err
	}

	username, err := resolveAddUsername(cmd, a, entries, title)
	if err != nil {
		return err
	}

	password, err := requiredPassword(cmd, a.prompter, "Password: ")
	if err != nil {
		return err
	}
	defer clear(password)

	url, err := stringField(cmd, a, "url", "URL (optional): ")
	if err != nil {
		return err
	}
	notes, err := stringField(cmd, a, "notes", "Notes (optional): ")
	if err != nil {
		return err
	}

	if _, err := v.Add(core.Entry{
		Title:    title,
		Username: username,
		Password: string(password),
		URL:      url,
		Notes:    notes,
	}); err != nil {
		return fmt.Errorf("adding entry: %w", err)
	}
	if err := v.SaveFile(a.vault); err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added %s (%s)\n", title, username)
	return err
}

// findDuplicate returns the entry that already has the same title and
// username. Titles compare case-insensitively; usernames compare exactly,
// because a different case is a different username. The vault itself allows
// duplicates; this is an add-time guard against accidental copies.
func findDuplicate(entries []core.Entry, title, username string) (core.Entry, bool) {
	i := slices.IndexFunc(entries, func(e core.Entry) bool {
		return strings.EqualFold(e.Title, title) && e.Username == username
	})
	if i < 0 {
		return core.Entry{}, false
	}
	return entries[i], true
}

// resolveAddUsername returns the username for a new entry. A --username flag is
// explicit, so a duplicate fails with the existing entry and an edit hint. A
// prompted username is re-asked until it no longer duplicates an entry with the
// same title.
func resolveAddUsername(cmd *cobra.Command, a *app, entries []core.Entry, title string) (string, error) {
	if cmd.Flags().Changed("username") {
		username, err := cmd.Flags().GetString("username")
		if err != nil {
			return "", err
		}
		if username = strings.TrimSpace(username); username == "" {
			return "", errors.New("username is required")
		}
		if e, ok := findDuplicate(entries, title, username); ok {
			return "", fmt.Errorf(
				"entry already exists: %q (%s); use a different username or run %q to update it",
				e.Title, e.Username, "gatekeep edit",
			)
		}
		return username, nil
	}

	for {
		username, err := requiredLine(cmd, a.prompter, "Username: ", "Username")
		if err != nil {
			return "", err
		}
		e, ok := findDuplicate(entries, title, username)
		if !ok {
			return username, nil
		}
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(),
			"%q already exists for %q; try another.\n", e.Username, e.Title); err != nil {
			return "", err
		}
	}
}

// stringField returns a flag value when the flag was provided, and prompts for
// an optional value otherwise.
func stringField(cmd *cobra.Command, a *app, flag, prompt string) (string, error) {
	if cmd.Flags().Changed(flag) {
		value, err := cmd.Flags().GetString(flag)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(value), nil
	}
	return optionalLine(a.prompter, prompt)
}
