package ws

import (
	"fmt"
	"io"
	"strings"
)

const (
	MAX_HEADER_SIZE  = 16 << 10 // 16KiB
	MAX_MESSAGE_SIZE = 16 << 20 // 16MiB

	WEB_SOCKET_GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11" // Server
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
	key, err := newSecWebsocketKey()
	if err != nil {
		return wrapError(ErrClientHandshake, err)
	}

	handshake := strings.Join([]string{
		"GET / HTTP/1.1",
		"Host: " + host,
		"Upgrade: websocket",
		"Connection: Upgrade",
		"Sec-WebSocket-Key: " + key,
		"Sec-WebSocket-Version: 13",
		"",
		"",
	}, "\r\n")

	if _, err := io.WriteString(socket, handshake); err != nil {
		return wrapError(ErrSocket, err)
	}

	response, err := readHeaders(socket)
	if err != nil {
		return wrapError(ErrClientHandshake, err)
	}

	headers := string(response)
	if !strings.HasPrefix(headers, "HTTP/1.1 101") {
		return fmt.Errorf("%w: %q", ErrClientHandshake, headers)
	}

	if !headerHasToken(headers, "Upgrade", "websocket") {
		return fmt.Errorf("%w: missing Upgrade: websocket", ErrClientHandshake)
	}
	if !headerHasToken(headers, "Connection", "Upgrade") {
		return fmt.Errorf("%w: missing Connection: Upgrade", ErrClientHandshake)
	}

	accept, ok := headerValue(headers, "Sec-WebSocket-Accept")
	if !ok {
		return fmt.Errorf("%w: missing Sec-WebSocket-Accept", ErrClientHandshake)
	}
	if expected := computeSecWebsocketAccept(key); accept != expected {
		return fmt.Errorf(
			"%w: Sec-WebSocket-Accept %q, want %q",
			ErrClientHandshake, accept, expected,
		)
	}

	return nil
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

func readMessage(socket Socket, mask bool) (Message, error) {
	var msg Message
	var last *MessageChunk
	var total int

	started := false
	for {
		frame, err := readFrame(socket, !mask)
		if err != nil {
			return Message{}, err
		}

		if isControl(frame.Opcode) {
			switch frame.Opcode {
			case OPCODE_CLOSE:
				_ = sendFrame(socket, true, OPCODE_CLOSE, frame.Payload, mask)
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
