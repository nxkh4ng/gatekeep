package cmd

import "github.com/spf13/cobra"

func newAddCmd(_ *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Add an entry",
		Args:  cobra.MaximumNArgs(1),
		RunE:  notImplemented,
	}
	cmd.Flags().String("username", "", "account username (prompted when omitted)")
	cmd.Flags().String("url", "", "entry URL (prompted when omitted)")
	cmd.Flags().String("notes", "", "entry notes (prompted when omitted)")
	return cmd
}
