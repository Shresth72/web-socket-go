package ws

import (
	"bytes"
	"fmt"
)

type Conn struct {
	Socket
	mask bool
}

func NewClient(socket Socket) *Conn {
	return &Conn{Socket: socket, mask: true}
}

func NewServer(socket Socket) *Conn {
	return &Conn{Socket: socket, mask: false}
}

// Message methods
func (c *Conn) SendMessage(kind MessageKind, payload []byte, chunkLen int) error {
	return sendMessage(c, kind, payload, chunkLen, c.mask)
}

func (c *Conn) ReadMessage() (Message, error) {
	return readMessage(c, c.mask)
}

// Frame methods
func (c *Conn) SendPing(payload []byte) error {
	if err := sendFrame(c, true, OPCODE_PING, payload, c.mask); err != nil {
		return err
	}

	frame, err := readFrame(c)
	if err != nil {
		return err
	}

	switch frame.Opcode {
	case OPCODE_PONG:
		if !bytes.Equal(frame.Payload, payload) {
			return fmt.Errorf(
				"%w: PONG payload %q does not match PING payload %q",
				ErrProtocol, frame.Payload, payload,
			)
		}
		return nil
	case OPCODE_CLOSE:
		_ = sendFrame(c, true, OPCODE_CLOSE, frame.Payload, c.mask)
		return ErrServerClose
	default:
		return fmt.Errorf(
			"%w: got opcode %#x while waiting for PONG",
			ErrUnexpectedFrame,
			frame.Opcode,
		)
	}
}

func (c *Conn) SendPong(payload []byte) error {
	return sendFrame(c, true, OPCODE_PONG, payload, c.mask)
}

func (c *Conn) SendClose(payload []byte) error {
	if len(payload) == 1 {
		return fmt.Errorf("ws: close payload must be empty or at least 2 bytes")
	}
	return sendFrame(c, true, OPCODE_CLOSE, payload, c.mask)
}
