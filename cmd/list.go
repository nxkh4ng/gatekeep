package cmd

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/nxkh4ng/gatekeep/core"
	"github.com/spf13/cobra"
)

func newListCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [query]",
		Short: "List entries",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, a, args)
		},
	}
	cmd.Flags().Bool("json", false, "print entries as JSON, including passwords")
	return cmd
}

func runList(cmd *cobra.Command, a *app, args []string) error {
	v, err := loadVault(cmd, a)
	if err != nil {
		return err
	}
	defer v.Lock()

	entries, err := v.Entries()
	if err != nil {
		return err
	}

	query := ""
	if len(args) == 1 {
		query = strings.TrimSpace(args[0])
	}
	if query != "" {
		needle := strings.ToLower(query)
		entries = slices.DeleteFunc(entries, func(e core.Entry) bool {
			return !strings.Contains(strings.ToLower(e.Title), needle) &&
				!strings.Contains(strings.ToLower(e.Username), needle)
		})
	}

	slices.SortFunc(entries, func(a, b core.Entry) int {
		if c := cmp.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)); c != 0 {
			return c
		}
		return cmp.Compare(strings.ToLower(a.Username), strings.ToLower(b.Username))
	})

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding entries: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return err
	}

	if len(entries) == 0 {
		message := "Vault is empty."
		if query != "" {
			message = "No matching entries."
		}
		_, err := fmt.Fprintln(cmd.ErrOrStderr(), message)
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "TITLE\tUSERNAME\tURL"); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", e.Title, e.Username, e.URL); err != nil {
			return err
		}
	}
	return w.Flush()
}
