package ws

import (
	"fmt"
	"io"
	"strings"
)

const (
	MAX_HEADER_SIZE  = 16 << 10 // 16KiB
	MAX_MESSAGE_SIZE = 16 << 20 // 16MiB

	SEC_WEBSOCKET_KEY = "dGhlIHNhbXBsZSBub25jZQ=="             // Client
	WEB_SOCKET_GUID   = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11" // Server
)

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

type MessageChunk struct {
	next    *MessageChunk
	Length  int
	Payload []byte
}

type Message struct {
	Kind   MessageKind
	Chunks []*MessageChunk
}

func ServerHandshake(socket Socket) error {
	request, err := readHeaders(socket)
	if err != nil {
		return wrapError(ErrProtocol, err)
	}

	secWebsocketKey, ok := headerValue(string(request), "Sec-WebSocket-Key")
	if !ok || !validSecWebsocketKey(secWebsocketKey) {
		return fmt.Errorf("%w: missing or invalid Sec-WebSocket-Key", ErrProtocol)
	}

	handshake := strings.Join([]string{
		"HTTP/1.1 101 Switching Protocols",
		"Upgrade: websocket",
		"Connection: Upgrade",
		"Sec-WebSocket-Accept: " + computeSecWebsocketAccept(secWebsocketKey),
		"",
		"",
	}, "\r\n")

	if _, err := io.WriteString(socket, handshake); err != nil {
		return wrapError(ErrSocket, err)
	}
	return nil
}

func ClientHandshake(socket Socket, host string) error {
	handshake := strings.Join([]string{
		"GET / HTTP/1.1",
		"Host: " + host,
		"Upgrade: websocket",
		"Connection: Upgrade",
		"Sec-WebSocket-Key: " + SEC_WEBSOCKET_KEY,
		"Sec-WebSocket-Version: 13",
		"",
		"",
	}, "\r\n")

	if _, err := io.WriteString(socket, handshake); err != nil {
		return err
	}

	response, err := readHeaders(socket)
	if err != nil {
		return wrapError(ErrClientHandshake, err)
	}

	if !strings.HasPrefix(string(response), "HTTP/1.1 101") {
		return fmt.Errorf("%w: %q", ErrClientHandshake, response)
	}

	return nil
}

func SendClientMessage(socket Socket, kind MessageKind, payload []byte, chunkLen int) error {
	return sendMessage(socket, kind, payload, chunkLen, true)
}

func SendServerMessage(socket Socket, kind MessageKind, payload []byte, chunkLen int) error {
	return sendMessage(socket, kind, payload, chunkLen, false)
}

func sendMessage(socket Socket, kind MessageKind, payload []byte, chunkLen int, mask bool) error {
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
			mask,
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

func ReadClientMessage(socket Socket) (Message, error) {
	return readMessage(socket, true)
}

func ReadServerMessage(socket Socket) (Message, error) {
	return readMessage(socket, false)
}

func readMessage(socket Socket, mask bool) (Message, error) {
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
				if err := sendFrame(socket, true, OPCODE_PONG, frame.Payload, mask); err != nil {
					return Message{}, wrapError(ErrSocket, err)
				}
			}
			continue
		}

		total += len(frame.Payload)
		if total > MAX_MESSAGE_SIZE {
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
