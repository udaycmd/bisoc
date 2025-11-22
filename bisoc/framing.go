package bisoc

import (
	"encoding/binary"
	"io"
	"math"
)

type frameHeader struct {
	fin     bool
	rsv1    bool
	rsv2    bool
	rsv3    bool
	masked  bool
	op      opcode
	maskKey [4]byte
	sz      uint64
}

func (fh *frameHeader) isControlFrame() bool {
	return fh.op >= 0x8 && fh.op <= 0xF
}

func (ws *wsConn) readFrameHeaderInto(fh *frameHeader) error {
	var err error
	buffer := make([]byte, 2)

	// read into buffer
	_, err = ws.rw.Reader.Read(buffer)
	if err != nil {
		return err
	}

	if ((buffer[0] >> 7) & 0x1) == 1 {
		fh.fin = true
	}

	fh.rsv1 = ((buffer[0] >> 6) & 0x1) == 1
	fh.rsv2 = ((buffer[0] >> 5) & 0x1) == 1
	fh.rsv3 = ((buffer[0] >> 4) & 0x1) == 1

	fh.op = opcode(buffer[0] & 0xF)

	if (buffer[1] >> 7) == 1 {
		fh.masked = true
	}

	payloadLen := uint64(buffer[1] & 0x7F)
	switch payloadLen {
	case 126:
		extLen := make([]byte, 2)
		_, err = ws.rw.Reader.Read(extLen)

		for i := range len(extLen) {
			fh.sz = (fh.sz << 8) | uint64(extLen[i])
		}

	case 127:
		extLen := make([]byte, 8)
		_, err = ws.rw.Reader.Read(extLen)

		for i := range len(extLen) {
			fh.sz = (fh.sz << 8) | uint64(extLen[i])
		}

	default:
		fh.sz = payloadLen
	}

	// All control frames MUST have a payload length of 125 bytes or less and MUST NOT be fragmented.
	if fh.isControlFrame() && (fh.sz > 125 || !fh.fin) {
		return errIllegalControlFrame
	}

	//	RSV1, RSV2, RSV3:  1 bit each
	//
	//	MUST be 0 unless an extension is negotiated that defines meanings for non-zero values.
	//	If a nonzero value is received and none of the negotiated extensions defines the
	//  meaning of such a nonzero value, the receiving endpoint MUST _Fail the WebSocket Connection_.
	if fh.rsv1 || fh.rsv2 || fh.rsv3 {
		return errFrameRsvBitsNotNegotiated
	}

	if fh.masked {
		// read the mask key
		if err == nil {
			_, err = ws.rw.Reader.Read(fh.maskKey[:])
		}
	}

	return err
}

// TODO: check if invoked as client and do masking of the frame
func (ws *wsConn) sendFrame(fh *frameHeader, payload []byte) error {
	var data byte
	var err error

	if fh.fin {
		data = byte(fh.op)
		data |= (1 << 7)
	}

	// send fin bit + op
	if err = ws.rw.Writer.WriteByte(data); err != nil {
		return err
	}

	data = 0
	if fh.sz < 126 {
		data |= byte(fh.sz)
		err = ws.rw.Writer.WriteByte(data)
	} else {
		var extLen []byte
		if fh.sz <= math.MaxUint16 {
			data |= 126
			extLen = make([]byte, 2)
			binary.BigEndian.PutUint16(extLen, uint16(fh.sz))
		} else if fh.sz > math.MaxUint16 {
			data |= 126
			extLen = make([]byte, 8)
			binary.BigEndian.PutUint64(extLen, uint64(fh.sz))
		}
		_, err = ws.rw.Writer.Write(append([]byte{data}, extLen...))
	}

	if err == nil {
		_, err = ws.rw.Writer.Write(payload)
		if err == nil {
			return ws.rw.Writer.Flush()
		}
	}

	return err
}

func (ws *wsConn) readPayloadChunkInto(buf []byte, fh *frameHeader, finishedPayloadLen uint64) (uint64, error) {
	remPayload := buf[finishedPayloadLen:]

	n, err := ws.rw.Reader.Read(remPayload)
	if err != nil && err != io.EOF {
		return 0, err
	}

	if fh.masked && n > 0 {
		var i uint64 = 0
		for ; i < uint64(n); i++ {
			maskIndex := (finishedPayloadLen + i) % 4
			remPayload[i] ^= fh.maskKey[maskIndex]
		}
	}

	return uint64(n), nil
}

func (ws *wsConn) readPayloadInto(buf []byte, fh *frameHeader) error {
	var finished uint64 = 0
	for finished < fh.sz {
		n, err := ws.readPayloadChunkInto(buf, fh, finished)
		if err != nil {
			return err
		}

		finished += uint64(n)
	}
	return nil
}
