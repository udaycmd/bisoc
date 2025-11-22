package bisoc

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"net/textproto"
	"slices"
	"strings"
)

func (ws *wsConn) checkClientHandshake(origins []string) error {
	tp := textproto.NewReader(ws.rw.Reader)

	rl, err := tp.ReadLine()
	if err != nil {
		return err
	}

	parts := strings.SplitN(rl, " ", 3)
	if len(parts) < 3 || parts[0] != "GET" {
		return errClientHandshake
	}

	headers, err := tp.ReadMIMEHeader()
	if err != nil {
		return errClientHandshake
	}

	if headers["Upgrade"][0] != "websocket" || headers["Host"][0] == "" || headers["Connection"][0] != "Upgrade" || headers["Sec-WebSocket-Version"][0] != "13" {
		return errClientHandshake
	}

	key := headers["Sec-WebSocket-Key"][0]
	if key == "" {
		return errClientHandshake
	}

	if len(origins) != 0 {
		origin := headers["Origin"][0]
		if origin != "" && !slices.Contains(origins, origin) {
			return errClientHandshake
		}
	}

	accept := computeAccept(key)

	respBuff := &bytes.Buffer{}
	respBuff.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	respBuff.WriteString("Connection: Upgrade\r\n")
	respBuff.WriteString("Upgrade: websocket\r\n")
	respBuff.WriteString("Sec-WebSocket-Accept: " + accept + "\r\n")
	respBuff.WriteString("\r\n")

	if _, err := ws.rw.Writer.Write(respBuff.Bytes()); err != nil {
		return err
	}

	return ws.rw.Writer.Flush()
}

func computeAccept(key string) string {
	const GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hasher := sha1.New()
	hasher.Write([]byte(key + GUID))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
