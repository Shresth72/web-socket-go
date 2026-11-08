package ws

import "fmt"

type Opcode = byte
type MessageKind struct{ opCode byte }

const (
	OPCODE_CONT  Opcode = 0x0
	OPCODE_TEXT  Opcode = 0x1
	OPCODE_BIN   Opcode = 0x2
	OPCODE_CLOSE Opcode = 0x8
	OPCODE_PING  Opcode = 0x9
	OPCODE_PONG  Opcode = 0xA
)

var (
	MESSAGE_TEXT = MessageKind{OPCODE_TEXT}
	MESSAGE_BIN  = MessageKind{OPCODE_BIN}
)

type Frame struct {
	Fin     bool
	Opcode  Opcode
	Payload []byte
}

type Message struct {
	Kind   MessageKind
	Chunks []MessageChunk
}

type MessageChunk struct {
	next    *MessageChunk
	Length  int
	Payload []byte
}

func isControl(opcode Opcode) bool {
	return 0x8 <= opcode && opcode <= 0xF
}

func IsControl(opcode Opcode) bool {
	return 0x8 <= opcode && opcode <= 0xF
}

func sendMessage(socket Socket, kind MessageKind, payload []byte, chunkLen int) error {
	if !(kind.opCode == OPCODE_TEXT || kind.opCode == OPCODE_BIN) {
		return fmt.Errorf("ws: invalid message kind %#x", kind.opCode)
	}
	if chunkLen <= 0 {
		return fmt.Errorf("ws: chunk length must be positive, got %d", chunkLen)
	}

	first := true
	for {
		frameLen := len(payload)
		if frameLen > chunkLen {
			frameLen = chunkLen
		}
		finish := frameLen == len(payload)

		frameOpcode := OPCODE_CONT
		if first {
			frameOpcode = kind.opCode
		}

		if err := sendFrame(
			socket,
			finish,
			frameOpcode,
			payload[:frameLen],
		); err != nil {
			return err
		}

		if finish {
			break
		}

		payload = payload[frameLen:]
		first = false
	}

	return nil
}
