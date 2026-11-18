package cmd

import (
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/nxkh4ng/gatekeep/core"
	"github.com/spf13/cobra"
)

func newShowCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <query>",
		Short: "Show an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, a, args[0])
		},
	}
	cmd.Flags().Bool("reveal", false, "print the password instead of masking it")
	return cmd
}

func runShow(cmd *cobra.Command, a *app, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return errors.New("query is required")
	}
	reveal, err := cmd.Flags().GetBool("reveal")
	if err != nil {
		return err
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
	if errors.Is(err, errAborted) {
		return printAborted(cmd)
	}
	if err != nil {
		return err
	}
	if err := printEntry(cmd, entry, reveal); err != nil {
		return err
	}
	if !reveal && entry.Password != "" {
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(),
			"Password hidden; rerun with --reveal flag to print it."); err != nil {
			return err
		}
	}
	return nil
}

// maskedPassword stands in for a hidden password. Its fixed length keeps the
// real password length out of the output.
const maskedPassword = "********"

// printEntry writes the entry fields to stdout, omitting empty optional ones.
// The password is masked unless reveal is true.
func printEntry(cmd *cobra.Command, e core.Entry, reveal bool) error {
	password := e.Password
	if !reveal && password != "" {
		password = maskedPassword
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Title:\t%s\n", e.Title)
	_, _ = fmt.Fprintf(w, "Username:\t%s\n", e.Username)
	_, _ = fmt.Fprintf(w, "Password:\t%s\n", password)
	if e.URL != "" {
		_, _ = fmt.Fprintf(w, "URL:\t%s\n", e.URL)
	}
	if e.Notes != "" {
		_, _ = fmt.Fprintf(w, "Notes:\t%s\n", e.Notes)
	}
	return w.Flush()
}
