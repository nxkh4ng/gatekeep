package cmd

import (
	"errors"
	"fmt"
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

	var username string
	if cmd.Flags().Changed("username") {
		username, err = cmd.Flags().GetString("username")
		if err != nil {
			return err
		}
		if username = strings.TrimSpace(username); username == "" {
			return errors.New("username is required")
		}
	} else {
		username, err = requiredLine(cmd, a.prompter, "Username: ", "Username")
		if err != nil {
			return err
		}
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
