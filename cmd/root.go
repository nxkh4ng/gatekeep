package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// appName is used in help text and for the default data directory.
const appName = "gatekeep"

// usageTemplate is cobra's default usage template with a single usage line for
// the root command. The template is inherited by subcommands, so they are
// handled explicitly to keep their default rendering.
const usageTemplate = `Usage:{{if not .HasParent}}
  {{.CommandPath}} [command] [flags]{{else}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// version is overridden at build time with -ldflags "-X .../cmd.version=...".
var version = "dev"

// versionFrom resolves the version reported by the binary. Build-time ldflags
// win, then the module version recorded in the build info, then "dev".
func versionFrom(ldflags string, info *debug.BuildInfo) string {
	if ldflags != "" && ldflags != "dev" {
		return ldflags
	}
	if info != nil {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

// resolvedVersion reads the running binary's build info and resolves its
// version. It is called when the command tree is built.
func resolvedVersion() string {
	info, _ := debug.ReadBuildInfo()
	return versionFrom(version, info)
}

// Execute runs the gatekeep CLI.
func Execute() error {
	vault, err := defaultVaultPath()
	if err != nil {
		return err
	}
	return newRoot(&app{vault: vault, prompter: newTerminalPrompter()}).Execute()
}

// newRoot builds the command tree with the given dependencies.
func newRoot(a *app) *cobra.Command {
	root := &cobra.Command{
		Use:           appName,
		Short:         "Local password manager",
		Long:          "gatekeep stores passwords in a single encrypted vault file.",
		Version:       resolvedVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	root.SetUsageTemplate(usageTemplate)
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
