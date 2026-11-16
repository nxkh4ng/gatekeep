package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/nxkh4ng/gatekeep/core"
	"github.com/spf13/cobra"
)

// loadVault prompts for the master password and returns the unlocked vault.
func loadVault(cmd *cobra.Command, a *app) (*core.Vault, error) {
	if _, err := os.Stat(a.vault); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("vault not found at %s; run gatekeep init", a.vault)
		}
		return nil, fmt.Errorf("checking vault: %w", err)
	}

	password, err := requiredPassword(cmd, a.prompter, "Master password: ")
	if err != nil {
		return nil, err
	}
	defer clear(password)

	v, err := core.LoadFile(a.vault, password)
	if err != nil {
		return nil, err
	}
	return v, nil
}
