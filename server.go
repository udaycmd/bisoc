package bisoc

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

type Server struct {
	// HandShakeTimeout is the duration for the handshake to complete.
	HandShakeTimeout time.Duration

	// This function checks which origins are permitted by Server.
	// If not provided, same origin policy will be applied.
	CheckOrigin func(r *http.Request) bool

	// Subprotocols specifies the server's supported protocols in order of
	// preference.
	Subprotocols []string
}

// Accept accepts a connection and upgrades it to a WebSocket Connection.
func (wss *Server) Accept(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	c, err := wss.upgrade(w, r)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// error generates http error response.
func (wss *Server) error(w http.ResponseWriter, code int, reason string) error {
	http.Error(w, http.StatusText(code), code)
	return errors.New(reason)
}

func (wss *Server) upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if r.Method != http.MethodGet {
		return nil, wss.error(w, http.StatusMethodNotAllowed, badHandShake+"request method is not GET")
	}

	if !headerContains(r.Header["Connection"], "upgrade") {
		return nil, wss.error(w, http.StatusBadRequest, badHandShake+"'Connection' header of the request does not contains 'upgrade'")
	}

	if r.Header.Get("Sec-Websocket-Version") != "13" {
		return nil, wss.error(w, http.StatusBadRequest, "unsupported websocket version")
	}

	if !headerContains(r.Header["Upgrade"], "websocket") {
		return nil, wss.error(w, http.StatusUpgradeRequired, badHandShake+"'Upgrade' header of request does not contains 'websocket'")
	}

	cors := wss.CheckOrigin
	if cors == nil {
		// checks for the same origin.
		cors = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if len(origin) == 0 {
				return true
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}

			return u.Host == r.Host
		}
	}

	if !cors(r) {
		return nil, wss.error(w, http.StatusForbidden, "request origin not allowed")
	}

	challengeKey := r.Header.Get("Sec-Websocket-Key")
	if !isChallengeKeyValid(challengeKey) {
		return nil, wss.error(w, http.StatusBadRequest, "'Sec-WebSocket-Key' header must be a base64-encoded 16-byte string")
	}

	subprotocol := wss.selectSubProtocol(r)

	rawConn, brw, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return nil, wss.error(w, http.StatusInternalServerError, "hijack error: "+err.Error())
	}

	// Cleanup! Close the network connection when returning an error.
	defer func() {
		if rawConn != nil {
			rawConn.Close()
		}
	}()

	if brw.Reader.Buffered() > 0 {
		// it is abnormal behaviour for client to send data before completion of the opening
		// handshake, we must report this as an error to client.
		return nil, wss.error(w, http.StatusBadRequest, "bisoc does not implement HTTP pipelining")
	}

	// Setup Read and Write buffers for the server side connection.
	var br *bufio.Reader
	if brw.Reader.Size() > minBufSize {
		// use the hijacked buffered reader.
		br = brw.Reader
	}

	var writeBuf []byte
	buf := brw.AvailableBuffer()
	if len(buf) > maxFrameHeaderSize+minBufSize {
		// use the hijacked write buffer.
		writeBuf = buf
	}

	c := newConn(rawConn, false, br, writeBuf)
	c.subprotocol = subprotocol

	respBuf := buf
	if len(c.writeBuf) > len(respBuf) {
		respBuf = c.writeBuf
	}

	// Reset the response buffer
	respBuf = respBuf[:0]

	// Write the handshake header in the response buffer
	respBuf = append(respBuf, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: "...)
	respBuf = append(respBuf, computeAcceptKey([]byte(challengeKey))...)
	respBuf = append(respBuf, "\r\n"...)
	if c.subprotocol != "" {
		respBuf = append(respBuf, "Sec-WebSocket-Protocol: "...)
		respBuf = append(respBuf, c.subprotocol...)
		respBuf = append(respBuf, "\r\n"...)
	}
	respBuf = append(respBuf, "\r\n"...)

	// Set a HandShakeTimeout deadline or clear any deadline configured by the http server.
	if wss.HandShakeTimeout > 0 {
		if err := rawConn.SetWriteDeadline(time.Now().Add(wss.HandShakeTimeout)); err != nil {
			return nil, err
		}
	} else {
		if err := rawConn.SetDeadline(time.Time{}); err != nil {
			return nil, err
		}
	}

	if _, err := rawConn.Write(respBuf); err != nil {
		return nil, err
	}

	// Remove the HandShakeTimeout deadline if applied.
	if wss.HandShakeTimeout > 0 {
		if err := rawConn.SetDeadline(time.Time{}); err != nil {
			return nil, err
		}
	}

	// Success! This stops the above deferred cleanup function from closing the connection.
	rawConn = nil
	return c, nil
}

func (wss *Server) selectSubProtocol(r *http.Request) string {
	if wss.Subprotocols != nil {
		clientProtocols := subProtocols(r)
		for _, cp := range clientProtocols {
			if slices.Contains(wss.Subprotocols, cp) {
				return cp
			}
		}
	}

	return ""
}

func subProtocols(r *http.Request) []string {
	header := strings.TrimSpace(r.Header.Get("Sec-Websocket-Protocol"))
	if header == "" {
		return nil
	}
	protocols := strings.Split(header, ",")
	for i := range protocols {
		protocols[i] = strings.TrimSpace(protocols[i])
	}
	return protocols
}

func headerContains(values []string, target string) bool {
	for i := range values {
		if strings.EqualFold(values[i], target) {
			return true
		}
	}

	return false
}

// Generate 'Sec-WebSocket-Accept' by concatenating the challengeKey with
// [KEY_GUID] and return the base64-encoded form of the sha1 hash of this
// concatenated string.
// Descibed in RFC 6455 (Section 1.3).
func computeAcceptKey(challengeKey []byte) string {
	sha := sha1.New()
	sha.Write(challengeKey)
	sha.Write(KEY_GUID)
	return base64.StdEncoding.EncodeToString(sha.Sum(nil))
}

// A |Sec-WebSocket-Key| header field with a base64-encoded
// (see Section 4 of [RFC4648]) value that, when decoded, is 16 bytes in length.
// Described in RFC 6455 (Section 4.2.1).
func isChallengeKeyValid(s string) bool {
	if s == "" {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	return err == nil && len(decoded) == 16
}
