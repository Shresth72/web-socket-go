package ws

import "fmt"

type Opcode = byte
type MessageKind struct{ opCode byte }

const maxMessageSize = 16 << 20 // 16MiB

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
	Chunks []*MessageChunk
}

type MessageChunk struct {
	next    *MessageChunk
	Length  int
	Payload []byte
}

func Test() bool {
	return true
}

func isControl(opcode Opcode) bool {
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

func readMessage(socket Socket) (Message, error) {
	var msg Message
	var last *MessageChunk
	var total int

	started := false
	for {
		frame, err := readFrame(socket)
		if err != nil {
			return Message{}, err
		}

		if isControl(frame.Opcode) {
			switch frame.Opcode {
			case OPCODE_CLOSE:
				return Message{}, ErrServerClose
			case OPCODE_PING:
				if err := sendFrame(socket, true, OPCODE_PONG, frame.Payload); err != nil {
					return Message{}, wrapError(ErrSocket, err)
				}
			}
			continue
		}

		total += len(frame.Payload)
		if total > maxMessageSize {
			return Message{}, wrapError(
				ErrProtocol,
				fmt.Errorf("ws: message too large than allowed limit: %d bytes", total),
			)
		}

		chunk := &MessageChunk{
			Length:  len(frame.Payload),
			Payload: frame.Payload,
		}

		if !started {
			if frame.Opcode != OPCODE_TEXT && frame.Opcode != OPCODE_BIN {
				return Message{}, fmt.Errorf(
					"ws: first data frame must be TEXT or BIN, got %#x",
					frame.Opcode,
				)
			}
			msg.Kind = MessageKind{frame.Opcode}
			started = true
		} else {
			if frame.Opcode != OPCODE_CONT {
				return Message{}, fmt.Errorf("ws: expected CONT, got %#x", frame.Opcode)
			}
			last.next = chunk
		}

		msg.Chunks = append(msg.Chunks, chunk)
		last = chunk

		if frame.Fin {
			return msg, nil
		}
	}
}
