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
	return &cobra.Command{
		Use:   "show <query>",
		Short: "Show an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, a, args[0])
		},
	}
}

func runShow(cmd *cobra.Command, a *app, query string) error {
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
	return printEntry(cmd, entry)
}

// printEntry writes the entry fields to stdout, omitting empty optional ones.
func printEntry(cmd *cobra.Command, e core.Entry) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Title:\t%s\n", e.Title)
	_, _ = fmt.Fprintf(w, "Username:\t%s\n", e.Username)
	_, _ = fmt.Fprintf(w, "Password:\t%s\n", e.Password)
	if e.URL != "" {
		_, _ = fmt.Fprintf(w, "URL:\t%s\n", e.URL)
	}
	if e.Notes != "" {
		_, _ = fmt.Fprintf(w, "Notes:\t%s\n", e.Notes)
	}
	return w.Flush()
}
