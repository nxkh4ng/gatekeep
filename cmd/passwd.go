package cmd

import "github.com/spf13/cobra"

func newPasswdCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "passwd",
		Short: "Change the master password",
		Args:  cobra.NoArgs,
		RunE:  notImplemented,
	}
}
