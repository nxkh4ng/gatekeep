package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", appName, version)
			return err
		},
	}
}
