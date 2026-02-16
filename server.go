package bisoc

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Server struct {
	handshakeTimeout time.Duration
	Subprotocols     []string
	CheckOrigin      func(r *http.Request) bool
}

func (wss *Server) Connect() (*Conn, error) {
	return nil, nil
}

func (wss *Server) error(w http.ResponseWriter, code int, reason string) error {
	http.Error(w, http.StatusText(code), code)
	return errors.New(reason)
}

func (wss *Server) tryUpgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if r.Method != http.MethodGet {
		return nil, wss.error(w, http.StatusMethodNotAllowed, badHandShake+"request method is not GET")
	}

	if r.Header.Get("Connection") != "upgrade" {
		return nil, wss.error(w, http.StatusBadRequest, badHandShake+"'Connection' header of the request does not contains 'upgrade'")
	}

	if r.Header.Get("Sec-Websocket-Version") != "13" {
		return nil, wss.error(w, http.StatusBadRequest, "unsupported websocket version")
	}

	if r.Header.Get("Upgrade") != "websocket" {
		return nil, wss.error(w, http.StatusUpgradeRequired, badHandShake+"'Upgrade' header of request does not contains 'websocket'")
	}

	w.Header().Set("Upgrade", "websocket")
	cof := wss.CheckOrigin
	if cof == nil {
		cof = func(r *http.Request) bool {
			origin := r.Header["Origin"]
			if len(origin) == 0 {
				return true
			}
			u, err := url.Parse(origin[0])
			if err != nil {
				return false
			}

			return u.Host == r.Host
		}
	}

	if !cof(r) {
		return nil, wss.error(w, http.StatusForbidden, "request origin not allowed")
	}

	challengeKey := r.Header.Get("Sec-Websocket-Key")
	if !isChallengeKeyValid(challengeKey) {
		return nil, wss.error(w, http.StatusBadRequest, "'Sec-WebSocket-Key' header must be base64-encoded 16-byte string")
	}

	tcpConn, brw, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return nil, wss.error(w, http.StatusInternalServerError, "hijack: "+err.Error())
	}

	// cleanup
	defer func() {
		if tcpConn != nil {
			_ = tcpConn.Close()
		}
	}()

	if wss.handshakeTimeout > 0 {
		if err := tcpConn.SetWriteDeadline(time.Now().Add(wss.handshakeTimeout)); err != nil {
			return nil, err
		}
	} else {
		if err := tcpConn.SetDeadline(time.Time{}); err != nil {
			return nil, err
		}
	}

	// Write all headers to tcpConn

	tcpConn = nil
	return nil, nil
}

func (wss *Server) selectSubProtocol(r *http.Request) string {
	if wss.Subprotocols != nil {
		clientProtocols := subProtocols(r)
		for _, cp := range clientProtocols {
			for _, sp := range wss.Subprotocols {
				if cp == sp {
					return cp
				}
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
