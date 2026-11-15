package core

import (
	"encoding/hex"
	"time"
)

// Entry is one item stored in a vault. Timestamps are UTC.
type Entry struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	URL      string    `json:"url,omitempty"`
	Notes    string    `json:"notes,omitempty"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

// newEntryID returns a random 128-bit identifier as lowercase hex.
func newEntryID() (string, error) {
	var b [16]byte
	if err := randomFill(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
