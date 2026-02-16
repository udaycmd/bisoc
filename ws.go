package bisoc

import "net"

type wsConn struct {
	conn     net.Conn
	isServer bool
}
