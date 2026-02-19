package bisoc

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"
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

func isControlFrame(i int) bool {
	return i == BinMsg || i == TextMsg
}

// Conn represents a WebSocket connection.
type Conn struct {
	conn        net.Conn
	client      bool
	subprotocol string
	writeBuf    []byte
	br          *bufio.Reader
	readLimit   int
	reader      io.Reader
}

// msgReader helps stream a connection with fragmented messages
type msgReader struct {
	c         *Conn
	totalRead int
	remain    int
	eof       bool
	mask      [4]byte
	maskPos   int
}

func (mr *msgReader) Read(p []byte) (int, error) {
	if mr.remain == 0 {
		if mr.eof {
			return 0, io.EOF
		}

		if err := mr.nextFrame(); err != nil {
			return 0, err
		}
	}

	// be within frame boundaries
	if len(p) > mr.remain {
		p = p[:mr.remain]
	}

	n, err := mr.c.br.Read(p)
	if n > 0 {
		mr.totalRead += n
		if mr.totalRead > mr.c.readLimit {
			mr.c.Close()
			return 0, &CloseError{Code: StatusMessageTooBig}
		}

		for i := range n {
			p[i] ^= mr.mask[mr.maskPos%4]
			mr.maskPos++
		}

		mr.remain -= n
	}

	return n, err
}

func (mr *msgReader) nextFrame() error {
	for {
		var header [2]byte
		_, err := io.ReadFull(mr.c.br, header[:])
		if err != nil {
			return err
		}

		opcode := (header[0] & 0x0F)

		if isControlFrame(int(opcode)) {
			// TODO: handle this properly
			mr.c.handleControlFrame()

			continue
		}

		if opcode != continuation {
			return &CloseError{Code: StatusProtocolError}
		}

		mr.eof = (header[0] & fin) == 1
		// TODO: Read payload len, mask key.
		return nil
	}
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
		conn:      conn,
		client:    isClient,
		br:        br,
		writeBuf:  writeBuf,
		readLimit: ReadLimit,
	}
}

// Sends a websocket message to the connected peer.
func (ws *Conn) SendMsg(msgKind int, data []byte) error {
	// TODO: construct the header
	return nil
}

// Receives a websocket message from the connected peer.
func (ws *Conn) RecvMsg() (int, []byte, error) {
	// clean leftovers
	if ws.reader != nil {
		_, err := io.Copy(io.Discard, ws.reader)
		if err != nil {
			return 0, nil, err
		}

		ws.reader = nil
	}

	return 0, nil, nil
}

func (ws *Conn) handleControlFrame() (int, []byte, error) {
	return 0, nil, nil
}

// Subprotocol returns the negotiated protocol for the connection.
func (ws *Conn) Subprotocol() string {
	return ws.subprotocol
}

func (ws *Conn) SetReadLimit(limit int) {
	ws.readLimit = limit
}

func (ws *Conn) SetDeadline(t time.Time) error {
	return ws.conn.SetDeadline(t)
}

func (ws *Conn) SetReadDeadline(t time.Time) error {
	return ws.conn.SetReadDeadline(t)
}

func (ws *Conn) SetWriteDeadline(t time.Time) error {
	return ws.conn.SetWriteDeadline(t)
}

func (ws *Conn) LocalAddr() net.Addr {
	return ws.conn.LocalAddr()
}

func (ws *Conn) RemoteAddr() net.Addr {
	return ws.conn.RemoteAddr()
}

// Close closes the underlying connection.
func (ws *Conn) Close() error {
	return ws.conn.Close()
}
