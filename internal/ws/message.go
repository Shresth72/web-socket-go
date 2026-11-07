package ws

type Opcode = byte
type MessageKind struct{ b byte }

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

func IsControl(opcode Opcode) bool {
	return 0x8 <= opcode && opcode <= 0xF
}

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
