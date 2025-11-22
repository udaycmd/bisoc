package bisoc

import (
	"bufio"
	"net"
)

type opcode byte

// The wsConn type represents a WebSocket connection.
type wsConn struct {
	raw      *net.Conn
	rw       *bufio.ReadWriter
	isServer bool
}

const (
	framePayloadMaxSize = 1024 * 1 // 1 Kilobyte
)

const (
	op_cont  opcode = 0x0
	op_text  opcode = 0x1
	op_bin   opcode = 0x2
	op_close opcode = 0x8
	op_ping  opcode = 0x9
	op_pong  opcode = 0xA
)

func (ws *wsConn) RawConn() *net.Conn {
	return ws.raw
}
