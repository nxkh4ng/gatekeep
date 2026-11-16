package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
)

// bodyVersion is the JSON schema version of the encrypted body.
const bodyVersion = 1

var (
	// ErrLocked reports an operation attempted on a vault after Lock.
	ErrLocked = errors.New("vault is locked")

	// ErrNotFound reports an entry ID that is not in the vault.
	ErrNotFound = errors.New("entry not found")
)

// vaultData is the plaintext JSON body of a vault file.
type vaultData struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// Vault is an unlocked password vault held in memory. A Vault is not safe for
// concurrent use; callers must serialize access.
type Vault struct {
	keys   vaultKeys
	data   vaultData
	locked bool
}

// New creates an empty, unlocked vault protected by password.
func New(password []byte) (*Vault, error) {
	return newVault(password, defaultKDF)
}

func newVault(password []byte, kdf kdfParams) (*Vault, error) {
	k, err := newKeys(password, kdf)
	if err != nil {
		return nil, err
	}
	return &Vault{
		keys: k,
		data: vaultData{Version: bodyVersion, Entries: []Entry{}},
	}, nil
}

// Open decrypts a vault file and returns the unlocked vault. A wrong password
// returns ErrInvalidPassword; a malformed body returns ErrCorrupt.
func Open(data, password []byte) (*Vault, error) {
	k, body, err := unlock(data, password)
	if err != nil {
		return nil, err
	}
	defer clear(body)

	var d vaultData
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, fmt.Errorf("%w: body is not valid JSON", ErrCorrupt)
	}
	if d.Version != bodyVersion {
		return nil, fmt.Errorf("%w: body version %d", ErrCorrupt, d.Version)
	}
	if d.Entries == nil {
		d.Entries = []Entry{}
	}
	return &Vault{keys: k, data: d}, nil
}

// Seal encrypts the current contents into a vault file. Each call uses fresh
// nonces, so the result differs from previous seals.
func (v *Vault) Seal() ([]byte, error) {
	if v.locked {
		return nil, ErrLocked
	}
	body, err := json.Marshal(v.data)
	if err != nil {
		return nil, fmt.Errorf("encoding vault body: %w", err)
	}
	defer clear(body)
	return v.keys.seal(body)
}

// ChangePassword replaces the master password and key material. The change
// takes effect on the next Seal.
func (v *Vault) ChangePassword(newPassword []byte) error {
	if v.locked {
		return ErrLocked
	}
	k, err := newKeys(newPassword, v.keys.kdf)
	if err != nil {
		return err
	}
	clear(v.keys.kek[:])
	clear(v.keys.dek[:])
	v.keys = k
	return nil
}

// Lock drops all entries and zeroes the key material. It is safe to call more
// than once; every other method returns ErrLocked afterwards.
func (v *Vault) Lock() {
	v.keys = vaultKeys{}
	v.data = vaultData{}
	v.locked = true
}

// Entries returns a copy of the vault's entries in insertion order.
func (v *Vault) Entries() ([]Entry, error) {
	if v.locked {
		return nil, ErrLocked
	}
	return slices.Clone(v.data.Entries), nil
}

// Entry returns the entry with the given ID.
func (v *Vault) Entry(id string) (Entry, error) {
	if v.locked {
		return Entry{}, ErrLocked
	}
	i := v.index(id)
	if i < 0 {
		return Entry{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return v.data.Entries[i], nil
}

// Add stores a new entry and returns its generated ID. Any ID and timestamps
// in e are overwritten.
func (v *Vault) Add(e Entry) (string, error) {
	if v.locked {
		return "", ErrLocked
	}
	id, err := newEntryID()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	e.ID = id
	e.Created = now
	e.Updated = now
	v.data.Entries = append(v.data.Entries, e)
	return id, nil
}

// Update replaces the entry with the given ID, keeping its ID and creation
// time. It returns ErrNotFound when the ID is unknown.
func (v *Vault) Update(id string, e Entry) error {
	if v.locked {
		return ErrLocked
	}
	i := v.index(id)
	if i < 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	e.ID = id
	e.Created = v.data.Entries[i].Created
	e.Updated = time.Now().UTC()
	v.data.Entries[i] = e
	return nil
}

// Delete removes the entry with the given ID.
func (v *Vault) Delete(id string) error {
	if v.locked {
		return ErrLocked
	}
	i := v.index(id)
	if i < 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	v.data.Entries = slices.Delete(v.data.Entries, i, i+1)
	return nil
}

// index returns the position of id in the entries, or -1.
func (v *Vault) index(id string) int {
	return slices.IndexFunc(v.data.Entries, func(e Entry) bool { return e.ID == id })
}
