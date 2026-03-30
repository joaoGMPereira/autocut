package handlers

import (
	"crypto/rand"
	"fmt"
)

// newJobID generates a random UUID-like identifier using only stdlib.
// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (hex groups)
func newJobID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// Set version 4 bits (RFC 4122)
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
