package cmd

import "github.com/spf13/cobra"

func newEditCmd(_ *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <query>",
		Short: "Edit an entry",
		Args:  cobra.ExactArgs(1),
		RunE:  notImplemented,
	}
	cmd.Flags().String("title", "", "new title")
	cmd.Flags().String("username", "", "new username")
	cmd.Flags().String("url", "", "new URL")
	cmd.Flags().String("notes", "", "new notes")
	cmd.Flags().Bool("password", false, "prompt for a new password")
	return cmd
}
