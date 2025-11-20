package bisoc

import (
	"bufio"
	"encoding/binary"
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
	Len     uint64
}

func (fh *frameHeader) readFrameHeaderInto(r *bufio.Reader) error {
	var err error
	buffer := make([]byte, 2)

	// read into buffer
	_, err = r.Read(buffer)
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
		_, err = r.Read(extLen)

		for i := range len(extLen) {
			fh.Len = (fh.Len << 8) | uint64(extLen[i])
		}
	case 127:
		extLen := make([]byte, 8)
		_, err = r.Read(extLen)

		for i := range len(extLen) {
			fh.Len = (fh.Len << 8) | uint64(extLen[i])
		}
	default:
		fh.Len = payloadLen
	}

	if fh.masked {
		// read the mask key
		if err == nil {
			_, err = r.Read(fh.maskKey[:])
		}
	}

	return err
}

func sendFrame(w *bufio.Writer, fh *frameHeader, payload []byte) error {
	var data byte
	var err error

	if fh.fin {
		data = byte(fh.op)
		data |= (1 << 7)
	}

	// send fin bit + op
	if err = w.WriteByte(data); err != nil {
		return err
	}

	data = 0
	if fh.Len < 126 {
		data |= byte(fh.Len)
		err = w.WriteByte(data)
	} else {
		var extLen []byte
		if fh.Len <= math.MaxUint16 {
			data |= 126
			extLen = make([]byte, 2)
			binary.BigEndian.PutUint16(extLen, uint16(fh.Len))
		} else if fh.Len > math.MaxUint16 {
			data |= 126
			extLen = make([]byte, 8)
			binary.BigEndian.PutUint64(extLen, uint64(fh.Len))
		}
		_, err = w.Write(append([]byte{data}, extLen...))
	}

	// TODO: check if invoked as client and do masking of the frame

	if err == nil {
		_, err = w.Write(payload)
		if err == nil {
			return w.Flush()
		}
	}

	return err
}
