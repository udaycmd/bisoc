package bisoc

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

// Generates a 16 bytes long base64-encoded string which will
// be echoed in the 'Sec-WebSocket-Key' header field in the client handshake.
func generateChallengeKey() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
