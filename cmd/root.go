package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// appName is used in help text and for the default data directory.
const appName = "gatekeep"

// version is overridden at build time with -ldflags "-X .../cmd.version=...".
var version = "dev"

// Execute runs the gatekeep CLI.
func Execute() error {
	vault, err := defaultVaultPath()
	if err != nil {
		return err
	}
	return newRoot(&app{vault: vault}).Execute()
}

// newRoot builds the command tree with the given dependencies.
func newRoot(a *app) *cobra.Command {
	root := &cobra.Command{
		Use:           appName,
		Short:         "Local password manager",
		Long:          "GateKeep stores passwords in a single encrypted vault file.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	root.SetVersionTemplate(appName + " {{.Version}}\n")
	root.PersistentFlags().StringVar(&a.vault, "vault", a.vault, "path to the vault file")

	root.AddCommand(
		newInitCmd(a),
		newAddCmd(a),
		newListCmd(a),
		newShowCmd(a),
		newEditCmd(a),
		newRmCmd(a),
		newPasswdCmd(a),
		newVersionCmd(a),
	)
	return root
}

// defaultVaultPath returns $XDG_DATA_HOME/gatekeep/vault.gk, falling back to
// ~/.local/share/gatekeep/vault.gk.
func defaultVaultPath() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" && filepath.IsAbs(dir) {
		return filepath.Join(dir, appName, "vault.gk"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", appName, "vault.gk"), nil
}
