package core

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID returns a random 16-character hex identifier. crypto/rand.Read does
// not fail on supported platforms, so its error is not handled.
func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
