package bisoc

// Globally Unique Identifier (GUID) used for generating
// 'Sec-WebSocket-Accept' by server during initial handshake.
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

const (
	// Frame header (0th byte) bit definations, described in RFC 6455 (Section 5.2)
	fin  = 1 << 7
	rsv1 = 1 << 6
	rsv2 = 1 << 5
	rsv3 = 1 << 4

	// Frame header (1st byte) bit defination, described in RFC 6455 (Section 5.2)
	ismasked = 1 << 7

	// All control frames MUST have a payload length of 125 bytes or less and MUST NOT be fragmented.
	maxControlFramePayloadSize = 125

	// Maximum frame header size in bytes i.e Fixed(2) + Payload Length(8) + Masking Key(4).
	maxFrameHeaderSize = 14

	// 0 denotes a continuation frame
	continuation = 0x0
)

const (
	badHandShake = "bisoc: illegal handshake by client: "

	// Minimum read and write buffer sizes
	MinBufSize = 512

	// Default Read buffer size
	ReadBufSize = 4096

	// Default Write buffer size
	WriteBufSize = 4096

	// Max Size of message payload (64 MB)
	ReadLimit = 67108864
)

// Connection Close Code Numbers as described in RFC 6455 (Section 11.7).
const (
	StatusNormalClosure           = 1000
	StatusGoingAway               = 1001
	StatusProtocolError           = 1002
	StatusUnsupportedData         = 1003
	StatusNoStatusReceived        = 1005
	StatusAbnormalClosure         = 1006
	StatusInvalidFramePayloadData = 1007
	StatusPolicyViolation         = 1008
	StatusMessageTooBig           = 1009
	StatusMandatoryExtension      = 1010
	StatusInternalServerError     = 1011
	StatusServiceRestart          = 1012
	StatusTryAgainLater           = 1013
	StatusTLSHandshake            = 1015
)
