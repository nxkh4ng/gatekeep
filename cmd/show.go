package cmd

import "github.com/spf13/cobra"

func newShowCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "show <query>",
		Short: "Show an entry",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented,
	}
}
