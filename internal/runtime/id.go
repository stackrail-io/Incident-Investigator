package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// newID returns a random, RFC-4122-ish identifier. We avoid a UUID dependency
// since a 16-byte random hex value is sufficient for in-memory identity.
func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read never fails on supported platforms, but stay defensive.
		panic(fmt.Sprintf("id generation failed: %v", err))
	}
	id := fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b[:]))
	return id
}
