package cmd

import "github.com/spf13/cobra"

func newListCmd(_ *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [query]",
		Short: "List entries",
		Args:  cobra.MaximumNArgs(1),
		RunE:  notImplemented,
	}
	cmd.Flags().Bool("json", false, "print entries as JSON, including passwords")
	return cmd
}
