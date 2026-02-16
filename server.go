package bisoc

import (
	"crypto/sha1"
	"encoding/base64"
)

// Generate 'Sec-WebSocket-Accept' by concatenating the challengeKey with
// [KEY_GUID] and return the base64-encoded form of the sha1 hash of this
// concatenated string.
// Descibed in RFC 6455 (Section 1.3).
func computeAcceptKey(challengeKey []byte) string {
	sha := sha1.New()
	sha.Write(challengeKey)
	sha.Write(KEY_GUID)
	return base64.StdEncoding.EncodeToString(sha.Sum(nil))
}

// A |Sec-WebSocket-Key| header field with a base64-encoded
// (see Section 4 of [RFC4648]) value that, when decoded, is 16 bytes in length.
// Described in RFC 6455 (Section 4.2.1).
func isChallengeKeyValid(s string) bool {
	if s == "" {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	return err == nil && len(decoded) == 16
}
