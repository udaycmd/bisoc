package bisoc

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
	"unicode/utf8"
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
	return i >= CloseMsg && i <= PongMsg
}

func isDataFrame(i int) bool {
	return i == TextMsg || i == BinMsg
}

func validateRSV(b byte) error {
	if b&(rsv1|rsv2|rsv3) != 0 {
		return &CloseError{Code: StatusProtocolError, Reason: "rsv bits are set but not negotiated"}
	}

	return nil
}

func isValidCloseCode(code int) bool {
	switch code {
	case StatusNormalClosure,
		StatusGoingAway,
		StatusProtocolError,
		StatusUnsupportedData,
		StatusInvalidFramePayloadData,
		StatusPolicyViolation,
		StatusMessageTooBig,
		StatusMandatoryExtension,
		StatusInternalServerError,
		StatusServiceRestart,
		StatusTryAgainLater:
		return true
	default:
		return code >= 3000 && code <= 4999
	}
}

// Conn represents a WebSocket connection.
type Conn struct {
	conn         net.Conn
	client       bool
	subprotocol  string
	writeBuf     []byte
	br           *bufio.Reader
	readLimit    int
	reader       io.Reader
	closeHandler func(int, string) error
	pingHandler  func(string) error
	pongHandler  func(string) error
}

// newConn creates a new WebSocket connection [Conn].
func newConn(conn net.Conn, isClient bool, br *bufio.Reader, writeBuf []byte) *Conn {
	if br == nil {
		br = bufio.NewReaderSize(conn, ReadBufSize)
	}

	if writeBuf == nil {
		writeBuf = make([]byte, WriteBufSize)
	}

	c := &Conn{
		conn:      conn,
		client:    isClient,
		br:        br,
		writeBuf:  writeBuf,
		readLimit: ReadLimit,
	}

	c.OnClose(nil)
	c.OnPing(nil)
	c.OnPong(nil)

	return c
}

// msgReader helps stream a connection with fragmented messages
type msgReader struct {
	c           *Conn
	totalRead   int
	remain      int
	eof         bool
	mask        []byte
	maskPos     int
	emptyFrames int
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
			return n, &CloseError{Code: StatusMessageTooBig}
		}

		if mr.mask != nil {
			for i := range n {
				p[i] ^= mr.mask[mr.maskPos&3]
				mr.maskPos++
			}
		}

		mr.remain -= n
	}

	return n, err
}

func (mr *msgReader) nextFrame() error {
	for {
		header, err := mr.c.readHeader(2)
		if err != nil {
			return err
		}

		if err := validateRSV(header[0]); err != nil {
			return err
		}

		opcode := int(header[0] & 0x0F)
		final := (header[0] & fin) != 0

		if isControlFrame(opcode) {
			if !final {
				return &CloseError{Code: StatusProtocolError, Reason: "fin bit not set in control frame"}
			}

			payload, err := mr.c.readPayload(header, true)
			if err != nil {
				return err
			}

			if err := mr.c.handleControlFrame(opcode, payload); err != nil {
				return err
			}

			continue
		}

		if opcode != continuation {
			return &CloseError{Code: StatusProtocolError, Reason: "continuation frame not sent"}
		}

		mr.eof = final
		l, mask, err := mr.c.readExtensions(header)
		if err != nil {
			return err
		}

		mr.remain, mr.mask, mr.maskPos = int(l), mask, 0
		if mr.remain == 0 && !mr.eof {
			mr.emptyFrames++
			if mr.emptyFrames > maxEmptyFrames {
				return &CloseError{Code: StatusProtocolError, Reason: "too many empty continuation frames"}
			}

			continue
		}

		return nil
	}
}

// Sends a websocket message to the connected peer.
func (ws *Conn) SendMsg(msgKind int, data string) error {
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

	for {
		// parse the first two bytes of frame header
		header, err := ws.readHeader(2)
		if err != nil {
			return 0, nil, err
		}

		if err := validateRSV(header[0]); err != nil {
			return 0, nil, err
		}

		final := (header[0] & fin) != 0
		opcode := int(header[0] & 0x0F)

		if isControlFrame(opcode) {
			if !final {
				return 0, nil, &CloseError{Code: StatusProtocolError, Reason: "fin bit not set in control frame"}
			}

			payload, err := ws.readPayload(header, true)
			if err != nil {
				return 0, nil, err
			}

			if err := ws.handleControlFrame(opcode, payload); err != nil {
				return 0, nil, err
			}

			continue
		}

		if !isDataFrame(opcode) {
			return 0, nil, &CloseError{Code: StatusProtocolError, Reason: "unexpected start frame"}
		}

		l, mask, err := ws.readExtensions(header)
		if err != nil {
			return 0, nil, err
		}

		ws.reader = &msgReader{
			c:      ws,
			eof:    final,
			remain: int(l),
			mask:   mask,
		}

		payload, err := io.ReadAll(ws.reader)
		ws.reader = nil
		if err != nil {
			return 0, nil, err
		}

		// RFC 6455 (Section 8.1)
		//
		// When an endpoint is to interpret a byte stream as UTF-8 but finds
		// that the byte stream is not, in fact, a valid UTF-8 stream, that
		// endpoint must fail the connection.
		//
		// Implementation Note: I am only validating this at message boundary
		// and not at every chunk/frame due to induced complexity of the procedure
		// which is infact is not strictly inforced by the standard.
		if opcode == TextMsg && !utf8.Valid(payload) {
			return 0, nil, &CloseError{
				Code:   StatusInvalidFramePayloadData,
				Reason: "invalid utf8 encoded text",
			}
		}

		return opcode, payload, nil
	}
}

// readHeader reads n bytes from the underlying connection
func (ws *Conn) readHeader(n int) ([]byte, error) {
	header := make([]byte, n)
	_, err := io.ReadFull(ws.br, header)
	return header, err
}

func (ws *Conn) readPayload(header []byte, control bool) ([]byte, error) {
	l, mask, err := ws.readExtensions(header)
	if err != nil {
		return nil, err
	}

	if control && l > maxControlFramePayloadSize {
		return nil, &CloseError{Code: StatusInvalidFramePayloadData, Reason: "control payload data too big"}
	}

	p := make([]byte, l)
	if _, err := io.ReadFull(ws.br, p); err != nil {
		return nil, err
	}

	if mask != nil {
		for i := range p {
			p[i] ^= mask[i&3]
		}
	}

	return p, nil
}

// readExtensions reads the extended header fields
func (ws *Conn) readExtensions(header []byte) (uint64, []byte, error) {
	l := uint64(header[1] & 0x7F)

	switch l {
	case 126:
		ext, err := ws.readHeader(2)
		if err != nil {
			return 0, nil, err
		}
		l = uint64(binary.BigEndian.Uint16(ext))
	case 127:
		ext, err := ws.readHeader(8)
		if err != nil {
			return 0, nil, err
		}
		l = binary.BigEndian.Uint64(ext)
	}

	if l > uint64(ws.readLimit) {
		return 0, nil, &CloseError{Code: StatusMessageTooBig}
	}

	ismasked := (header[1] & masked) != 0

	if !ws.client && !ismasked {
		return 0, nil, &CloseError{Code: StatusProtocolError, Reason: "client must mask all frames that it sends to the server"}
	}

	if ismasked {
		if ws.client {
			return 0, nil, &CloseError{Code: StatusProtocolError, Reason: "server must not mask any frames that it sends to the client"}
		}

		mask, err := ws.readHeader(4)
		if err != nil {
			return 0, nil, err
		}

		return l, mask, nil
	}

	return l, nil, nil
}

// handleControlFrame handles subsequent procedures when a specific
// control message arrives in the connection
func (ws *Conn) handleControlFrame(opcode int, p []byte) error {
	switch opcode {
	case CloseMsg:
		// RFC 6455 (Section 5.5.1)
		//
		// If there is a body, the first two bytes of the body MUST be a
		// 2-byte unsigned integer (in network byte order) representing a
		// status code defined in Section 7.4. Following the 2-byte integer,
		// the body MAY contain UTF-8-encoded data indicating the reason.
		code := noCode
		reason := ""

		l := len(p)
		if l == 1 {
			return &CloseError{Code: StatusProtocolError, Reason: "close message body too short"}
		}

		if l >= 2 {
			code = int(binary.BigEndian.Uint16(p[:2]))
			if !isValidCloseCode(code) {
				return &CloseError{Code: StatusProtocolError, Reason: "invalid close code sent"}
			}

			if l > 2 {
				if !utf8.Valid(p[2:]) {
					return &CloseError{Code: StatusProtocolError, Reason: "invalid utf8-encoded application data in close message body"}
				}

				reason = string(p[2:])
			}
		}

		ws.closeHandler(code, string(p))
		return &CloseError{Code: code, Reason: reason}
	case PingMsg:
		return ws.pingHandler(string(p))
	default:
		return ws.pongHandler(string(p))
	}
}

func (ws *Conn) OnClose(f func(code int, body string) error) {
	if f == nil {
		f = func(_ int, body string) error {
			// RFC 6455 (Section 5.5.1)
			//
			// If an endpoint receives a Close frame and did not previously send a
			// Close frame, the endpoint MUST send a Close frame in response.
			// (When sending a Close frame in response, the endpoint typically
			// echos the status code it received.)
			return ws.SendMsg(CloseMsg, body)
		}
	}

	ws.closeHandler = f
}

// attaches a pingHandler to the connection, default behaviour
// is to send a Pong frame with same appData in response.
func (ws *Conn) OnPing(f func(appData string) error) {
	if f == nil {
		f = func(appData string) error {
			return ws.SendMsg(PongMsg, appData)
		}
	}

	ws.pingHandler = f
}

// attaches a pongHandler to the connection, default behaviour
// is to do nothing (unsolicited pong frames)
func (ws *Conn) OnPong(f func(appData string) error) {
	if f == nil {
		f = func(_ string) error { return nil }
	}

	ws.pongHandler = f
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

// Close closes the underlying tcp connection.
func (ws *Conn) Close() error {
	return ws.conn.Close()
}
