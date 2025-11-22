package bisoc

import "errors"

var (
	errClientHandshake           = errors.New("client handshake rejected")
	errServerHandshake           = errors.New("server handshake rejected")
	errInvalidMsgKind            = errors.New("incorrect message type")
	errUnexpectedOpcode          = errors.New("unexpected opcode received")
	errIllegalControlFrame       = errors.New("illegal control frame")
	errFrameRsvBitsNotNegotiated = errors.New("rsv bits not negotiated")
)
