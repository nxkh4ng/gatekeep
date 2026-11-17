package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/nxkh4ng/gatekeep/core"
	"github.com/spf13/cobra"
)

// errAborted reports that the user cancelled an interactive prompt.
var errAborted = errors.New("aborted")

// printAborted reports a cancelled interactive prompt as a clean exit.
func printAborted(cmd *cobra.Command) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
	return err
}

// resolveEntry finds the entry matching query: an exact title first, then a
// title/username substring. Several matches open a picker.
func resolveEntry(cmd *cobra.Command, p Prompter, entries []core.Entry, query string) (core.Entry, error) {
	matches := slices.DeleteFunc(slices.Clone(entries), func(e core.Entry) bool {
		return !strings.EqualFold(e.Title, query)
	})
	if len(matches) == 0 {
		needle := strings.ToLower(query)
		matches = slices.DeleteFunc(slices.Clone(entries), func(e core.Entry) bool {
			return !strings.Contains(strings.ToLower(e.Title), needle) &&
				!strings.Contains(strings.ToLower(e.Username), needle)
		})
	}

	switch len(matches) {
	case 0:
		return core.Entry{}, fmt.Errorf("no entry matches %q", query)
	case 1:
		return matches[0], nil
	default:
		return pickEntry(cmd, p, matches)
	}
}

// pickEntry lists the matches and asks the user to choose one.
func pickEntry(cmd *cobra.Command, p Prompter, matches []core.Entry) (core.Entry, error) {
	slices.SortFunc(matches, func(a, b core.Entry) int {
		if c := cmp.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)); c != 0 {
			return c
		}
		return cmp.Compare(strings.ToLower(a.Username), strings.ToLower(b.Username))
	})

	w := tabwriter.NewWriter(cmd.ErrOrStderr(), 0, 4, 2, ' ', 0)
	for i, e := range matches {
		_, _ = fmt.Fprintf(w, "%d)\t%s\t%s\n", i+1, e.Title, e.Username)
	}
	if err := w.Flush(); err != nil {
		return core.Entry{}, err
	}

	prompt := fmt.Sprintf("Select [1-%d, q]: ", len(matches))
	for {
		line, err := p.Line(prompt)
		if err != nil {
			return core.Entry{}, err
		}
		if line = strings.TrimSpace(line); line == "" || strings.EqualFold(line, "q") {
			return core.Entry{}, errAborted
		}
		if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(matches) {
			return matches[n-1], nil
		}
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Invalid selection."); err != nil {
			return core.Entry{}, err
		}
	}
}
