package bisoc

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

var safeRandom = rand.Reader

// Generates a 16 bytes long base64-encoded string which will
// be echoed in the 'Sec-WebSocket-Key' header field in the client handshake.
func generateChallengeKey() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(safeRandom, b); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

// creates a 32 bit masking key for a client peer
func newMaskKey() [4]byte {
	var key [4]byte
	io.ReadFull(safeRandom, key[:])
	return key
}
