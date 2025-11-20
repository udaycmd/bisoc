package bisoc

import "errors"

type bisocError interface {
	error
}

var (
	errClientHandshake bisocError = errors.New("client handshake rejected")
	errServerHandshake bisocError = errors.New("server handshake rejected")
	errInvalidMsgKind  bisocError = errors.New("incorrect message type")
)
