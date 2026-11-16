package cmd

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

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
	cmd.Flags().Bool("json", false, "print entries as JSON without passwords")
	cmd.Flags().Bool("show-passwords", false, "include passwords in JSON output (requires --json)")
	return cmd
}

// listEntry is the JSON representation of an entry in list --json. Password is
// omitted unless --show-passwords is given.
type listEntry struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Username string    `json:"username"`
	Password string    `json:"password,omitempty"`
	URL      string    `json:"url,omitempty"`
	Notes    string    `json:"notes,omitempty"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

func runList(cmd *cobra.Command, a *app, args []string) error {
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	showPasswords, err := cmd.Flags().GetBool("show-passwords")
	if err != nil {
		return err
	}
	if showPasswords && !jsonOutput {
		return errors.New("--show-passwords requires --json")
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

	if jsonOutput {
		return printListJSON(cmd, entries, showPasswords)
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

// printListJSON writes entries as indented JSON. Passwords are included only
// when showPasswords is true.
func printListJSON(cmd *cobra.Command, entries []core.Entry, showPasswords bool) error {
	out := make([]listEntry, 0, len(entries))
	for _, e := range entries {
		le := listEntry{
			ID:       e.ID,
			Title:    e.Title,
			Username: e.Username,
			URL:      e.URL,
			Notes:    e.Notes,
			Created:  e.Created,
			Updated:  e.Updated,
		}
		if showPasswords {
			le.Password = e.Password
		}
		out = append(out, le)
	}

	// #nosec G117 -- passwords are included only with flag `--show-passwords`.
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding entries: %w", err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}
