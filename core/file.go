package core

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	// backupSuffix is appended to a vault path for the previous version.
	backupSuffix = ".bak"

	maxFileSize = bodyOffset + maxBodySize
)

// LoadFile opens the vault file at path and unlocks it with password.
func LoadFile(path string, password []byte) (*Vault, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening vault: %w", err)
	}
	defer func() { _ = f.Close() }()

	if fi, err := f.Stat(); err != nil {
		return nil, fmt.Errorf("checking vault size: %w", err)
	} else if fi.Size() > maxFileSize {
		return nil, fmt.Errorf("%w: file is %d bytes", ErrTooLarge, fi.Size())
	}

	data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading vault: %w", err)
	}
	if len(data) > maxFileSize {
		return nil, fmt.Errorf("%w: file is %d bytes", ErrTooLarge, len(data))
	}
	return Open(data, password)
}

// SaveFile seals the vault and atomically replaces path. The previous version
// is kept at path+".bak".
func (v *Vault) SaveFile(path string) error {
	data, err := v.Seal()
	if err != nil {
		return err
	}
	return writeVault(path, data)
}

// writeVault writes data to a temporary file next to path, fsyncs it, and
// renames it into place. The previous file, if any, is renamed to path+".bak"
// immediately before the new one takes over.
func writeVault(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating vault directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".gatekeep-*")
	if err != nil {
		return fmt.Errorf("creating temporary vault: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temporary vault: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("syncing temporary vault: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temporary vault: %w", err)
	}

	backup := path + backupSuffix
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, backup); err != nil {
			return fmt.Errorf("backing up vault: %w", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("checking existing vault: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replacing vault (previous version at %s): %w", backup, err)
	}

	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("opening vault directory: %w", err)
	}
	defer func() { _ = d.Close() }()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("syncing vault directory: %w", err)
	}
	return nil
}
