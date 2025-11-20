package bisoc

import (
	"bufio"
	"net"
)

type opcode byte

const (
	op_cont  opcode = 0x0
	op_text  opcode = 0x1
	op_bin   opcode = 0x2
	op_close opcode = 0x8
	op_ping  opcode = 0x9
	op_pong  opcode = 0xA
)

const (
	fragmentMaxSize = 1024 * 1 // 1 Kilobyte
)

type Bisoc struct {
	conn *net.TCPConn
	rw   *bufio.ReadWriter
}

func BisocCreate(tcpConn *net.TCPConn) *Bisoc {
	return &Bisoc{
		conn: tcpConn,
		rw:   bufio.NewReadWriter(bufio.NewReader(tcpConn), bufio.NewWriter(tcpConn)),
	}
}
