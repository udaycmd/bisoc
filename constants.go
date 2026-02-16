package bisoc

// Globally Unique Identifier (GUID) used for generating
// 'Sec-WebSocket-Accept' during initial handshake.
// Described in RFC 6455 (Section 1.3).
var KEY_GUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

const (
	// TextMsg is a text data message. Always Interpreted as UTF-8 encoded text.
	TextMsg = 0x1

	// BinMsg is a binary data message.
	BinMsg = 0x2

	// BinMsg and TextMsg are described in RFC 6455 (Section 5.6).

	// CloseMsg is a close control message.
	// If there is a body, the first two bytes of the body MUST be a 2-byte unsigned integer (in network byte order)
	// representing a status code.
	// Described in RFC 6455 (Section 5.5.1).
	CloseMsg = 0x8

	// PingMsg is a ping control message.
	// Described in RFC 6455 (Section 5.5.2).
	PingMsg = 0x9

	// PongMsg is a pong control message.
	// Described in RFC 6455 (Section 5.5.3).
	PongMsg = 0xA
)
