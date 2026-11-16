package cmd

import "github.com/spf13/cobra"

func newRmCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <query>",
		Short: "Delete an entry",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented,
	}
}
