package bisoc

type message struct {
	kind    opcode
	payload []byte
}

func (ws *wsConn) sendMsg(payload []byte, msgType opcode) error {
	if msgType != op_bin && msgType != op_text {
		return errInvalidMsgKind
	}
	payloadLen := len(payload)
	isFirst := true

	for {
		var op opcode
		framePayloadLen := min(payloadLen, framePayloadMaxSize)

		fin := (payloadLen - framePayloadLen) == 0
		if isFirst {
			op = msgType
		} else {
			op = op_cont
		}

		if err := ws.sendFrame(&frameHeader{fin: fin, op: op, sz: uint64(framePayloadLen)}, payload[:framePayloadLen]); err != nil {
			return err
		}

		payload = payload[framePayloadLen:]
		payloadLen -= framePayloadLen
		isFirst = false

		if payloadLen <= 0 {
			break
		}
	}

	return nil
}

func (ws *wsConn) readMsgInto(msg *message) error {
	isFirstFrame := true

	for {
		fh := &frameHeader{}
		if err := ws.readFrameHeaderInto(fh); err != nil {
			return err
		}

		if fh.isControlFrame() {
			switch fh.op {
			case op_ping:
				//	NOTE: A Ping frame may serve either as a keepalive or as a means to verify that the remote
				//  endpoint is still responsive.
				if err := ws.readPayloadInto(msg.payload, fh); err != nil {
					return err
				}

				if err := ws.sendFrame(&frameHeader{fin: true, op: op_pong, sz: uint64(len(msg.payload))},msg. payload); err != nil {
					return err
				}

			case op_pong:
				// A Pong frame MAY be sent unsolicited.  This serves as a unidirectional heartbeat.
				// A response to an unsolicited Pong frame is not expected.
				if err := ws.readPayloadInto(msg.payload, fh); err != nil {
					return err
				}

			case op_close:
				// TODO
				break

			default:
				return errUnexpectedOpcode
			}
		} else {
			if isFirstFrame {
				switch fh.op {
				case op_bin, op_text:
					msg.kind = fh.op
				default:
					return errUnexpectedOpcode
				}
				isFirstFrame = false
			} else {
				if fh.op != op_cont {
					return errUnexpectedOpcode
				}
			}
		}

		if fh.fin {
			break
		}
	}

	return nil
}
