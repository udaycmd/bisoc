package bisoc

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"net/textproto"
	"slices"
	"strings"
)

func (bisoc *Bisoc) checkClientHandshake(origins []string) error {
	tp := textproto.NewReader(bisoc.rw.Reader)

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

	if _, err := bisoc.rw.Writer.Write(respBuff.Bytes()); err != nil {
		return err
	}

	return bisoc.rw.Writer.Flush()
}

func computeAccept(key string) string {
	const GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hasher := sha1.New()
	hasher.Write([]byte(key + GUID))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

func (bisoc *Bisoc) sendMsg(payload []byte, msgType opcode) error {
	if msgType != op_bin && msgType != op_text {
		return errInvalidMsgKind
	}
	payloadLen := len(payload)
	isFirst := true

	for {
		var op opcode
		frameLen := min(payloadLen, fragmentMaxSize)

		fin := (payloadLen - frameLen) == 0
		if isFirst {
			op = msgType
		} else {
			op = op_cont
		}

		if err := sendFrame(bisoc.rw.Writer, &frameHeader{fin: fin, op: op, Len: uint64(frameLen)}, payload[:frameLen]); err != nil {
			return nil
		}

		payload = payload[frameLen:]
		payloadLen -= frameLen
		isFirst = false

		if payloadLen <= 0 {
			break
		}
	}

	return nil
}

func (bisoc *Bisoc) readMsg() error {

	for {
		fh := &frameHeader{}
		if err := fh.readFrameHeaderInto(bisoc.rw.Reader); err != nil {
			return err
		}

		break
	}
}
