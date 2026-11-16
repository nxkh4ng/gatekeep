package cmd

import "github.com/spf13/cobra"

func newInitCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a new vault",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
}
