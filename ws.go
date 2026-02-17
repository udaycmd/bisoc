package bisoc

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

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

// ReadMessage blocks until a message is received.
// It handles control frames (Ping, Pong) internally.
// It returns the message type (TextMsg or BinMsg), the payload, and an error.
// If a Close control frame is received, it returns an error.
func (c *WSConn) ReadMessage() (int, []byte, error) {
	var payloadBuffer []byte
	var initialOpCode int

	for {
		// Read the first two bytes of the header
		header := make([]byte, 2)
		if _, err := io.ReadFull(c.br, header); err != nil {
			return 0, nil, err
		}

		final := header[0]&0x80 != 0
		opcode := int(header[0] & 0x0F)
		masked := header[1]&0x80 != 0
		payloadLen := int64(header[1] & 0x7F)

		if payloadLen == 126 {
			extended := make([]byte, 2)
			if _, err := io.ReadFull(c.br, extended); err != nil {
				return 0, nil, err
			}
			payloadLen = int64(binary.BigEndian.Uint16(extended))
		} else if payloadLen == 127 {
			extended := make([]byte, 8)
			if _, err := io.ReadFull(c.br, extended); err != nil {
				return 0, nil, err
			}
			payloadLen = int64(binary.BigEndian.Uint64(extended))
		}

		var maskKey []byte
		if masked {
			maskKey = make([]byte, 4)
			if _, err := io.ReadFull(c.br, maskKey); err != nil {
				return 0, nil, err
			}
		}

		payload := make([]byte, payloadLen)
		if payloadLen > 0 {
			if _, err := io.ReadFull(c.br, payload); err != nil {
				return 0, nil, err
			}
		}

		if masked {
			for i := range payload {
				payload[i] ^= maskKey[i%4]
			}
		}

		switch opcode {
		case CloseMsg:
			return 0, nil, io.EOF
		case PingMsg:
			if err := c.WriteControl(PongMsg, payload); err != nil {
				return 0, nil, err
			}
			continue
		case PongMsg:
			continue
		case TextMsg, BinMsg:
			if initialOpCode != 0 {
				return 0, nil, errors.New("received new message start while waiting for continuation")
			}
			if final {
				return opcode, payload, nil
			}
			initialOpCode = opcode
			payloadBuffer = append(payloadBuffer, payload...)
		case 0: // Continuation Frame
			if initialOpCode == 0 {
				return 0, nil, errors.New("received continuation frame without valid start")
			}
			payloadBuffer = append(payloadBuffer, payload...)
			if final {
				return initialOpCode, payloadBuffer, nil
			}
		default:
			// Reserved/Unknown opcode
			return 0, nil, errors.New("unknown opcode")
		}
	}
}

// WriteMessage sends a message to the peer.
// It ensures thread safety for concurrent writers.
// data is the payload. msgType should be TextMsg or BinMsg.
func (c *WSConn) WriteMessage(msgType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// Frame header construction
	b0 := byte(0x80) | byte(msgType) // Setting FIN bit
	b1 := byte(0)
	if c.isClient {
		b1 |= 0x80
	}

	length := len(data)
	var header []byte

	switch {
	case length <= 125:
		b1 |= byte(length)
		header = []byte{b0, b1}
	case length <= 65535:
		b1 |= 126
		header = make([]byte, 4)
		header[0] = b0
		header[1] = b1
		binary.BigEndian.PutUint16(header[2:], uint16(length))
	default:
		b1 |= 127
		header = make([]byte, 10)
		header[0] = b0
		header[1] = b1
		binary.BigEndian.PutUint64(header[2:], uint64(length))
	}

	if _, err := c.conn.Write(header); err != nil {
		return err
	}

	if c.isClient {
		maskKey := make([]byte, 4)
		binary.LittleEndian.PutUint32(maskKey, c.rng.Uint32())
		if _, err := c.conn.Write(maskKey); err != nil {
			return err
		}

		maskedData := make([]byte, len(data))
		copy(maskedData, data)
		for i := range maskedData {
			maskedData[i] ^= maskKey[i%4]
		}
		if _, err := c.conn.Write(maskedData); err != nil {
			return err
		}
	} else {
		if _, err := c.conn.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// WriteControl writes a control message (Ping, Pong, Close).
func (c *WSConn) WriteControl(msgType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	b0 := byte(0x80) | byte(msgType)
	b1 := byte(len(data))
	if c.isClient {
		b1 |= 0x80
	}

	header := []byte{b0, b1}
	if _, err := c.conn.Write(header); err != nil {
		return err
	}

	if c.isClient {
		maskKey := make([]byte, 4)
		binary.LittleEndian.PutUint32(maskKey, c.rng.Uint32())

		if _, err := c.conn.Write(maskKey); err != nil {
			return err
		}

		maskedData := make([]byte, len(data))
		copy(maskedData, data)
		for i := range maskedData {
			maskedData[i] ^= maskKey[i%4]
		}
		if _, err := c.conn.Write(maskedData); err != nil {
			return err
		}
	} else {
		if _, err := c.conn.Write(data); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the underlying connection.
func (c *WSConn) Close() error {
	c.WriteControl(CloseMsg, []byte{})
	return c.conn.Close()
}
