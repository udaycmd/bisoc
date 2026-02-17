package bisoc

import (
	"bufio"
	"fmt"
	"net"
)

type CloseError struct {
	Code   int    // Status code of error
	Reason string // Optional Reason for error
}

func (e *CloseError) Error() string {
	s := fmt.Sprintf("websocket closed with code: %d", e.Code)
	switch e.Code {
	case StatusNormalClosure:
		s += " normal"
	case StatusGoingAway:
		s += " going-away"
	case StatusProtocolError:
		s += " protocol-error"
	case StatusUnsupportedData:
		s += " unsupported-data"
	case StatusNoStatusReceived:
		s += " no-status"
	case StatusAbnormalClosure:
		s += " abnormal-closure"
	case StatusInvalidFramePayloadData:
		s += " invalid-payload-data"
	case StatusPolicyViolation:
		s += " policy-violation"
	case StatusMessageTooBig:
		s += " message-too-big"
	case StatusMandatoryExtension:
		s += " mandatory extension missing"
	case StatusInternalServerError:
		s += " internal-server-error"
	case StatusTLSHandshake:
		s += " tls-error"
	}
	if e.Reason != "" {
		s += ": " + e.Reason
	}

	return s
}

// Conn represents a WebSocket connection.
type Conn struct {
	conn        net.Conn
	client      bool
	subprotocol string
	writeBuf    []byte
	br          *bufio.Reader
}

// newConn creates a new WebSocket connection [Conn].
func newConn(conn net.Conn, isClient bool, br *bufio.Reader, writeBuf []byte) *Conn {
	if br == nil {
		br = bufio.NewReaderSize(conn, ReadBufSize)
	}

	if writeBuf == nil {
		writeBuf = make([]byte, WriteBufSize)
	}

	return &Conn{
		conn:     conn,
		client:   isClient,
		br:       br,
		writeBuf: writeBuf,
	}
}

// Sends a websocket message to the connected peer.
func SendMsg(msgKind int, data []byte) error {
	return nil
}

// Receives a websocket message from the connected peer.
func RecvMsg() (int, []byte, error) {
	return 0, nil, nil
}
