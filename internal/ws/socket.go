package ws

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
)

type Socket interface {
	io.Reader
	io.Writer
}

func clientHandshake(socket Socket, host []byte) error {
	handshake := strings.Join([]string{
		"GET / HTTP/1.1",
		"Host: " + string(host),
		"Upgrade: websocket",
		"Connection: Upgrade",
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==",
		"Sec-WebSocket-Version: 13",
		"",
		"",
	}, "\r\n")

	if _, err := io.WriteString(socket, handshake); err != nil {
		return err
	}

	// Read until "\r\n\r\n"
	var buffer [1024]byte
	complete := false
	n := 0
	for n < len(buffer) {
		m, err := socket.Read(buffer[n:])
		n += m
		if err != nil {
			return err
		}
		if bytes.Contains(buffer[:n], []byte("\r\n\r\n")) {
			complete = true
			break
		}
	}
	if !complete {
		return fmt.Errorf("%w: incomplete response headers", ErrClientHandshake)
	}

	response := string(buffer[:n])
	if !strings.HasPrefix(response, "HTTP/1.1 101") {
		return fmt.Errorf("%w: %q", ErrClientHandshake, response)
	}

	return nil
}

func sendFrame(socket Socket, fin bool, opcode Opcode, payload []byte) error {

	// Send FIN and OPCODE
	data := opcode & 0x0F
	if fin {
		data |= 0x80 // 1<<7
	}

	if _, err := socket.Write([]byte{data}); err != nil {
		return wrapError(ErrSocket, err)
	}

	// Send MASK bit and Payload length
	n := len(payload)
	switch {
	case n <= 125:
		data := 0x80 | byte(n)
		if _, err := socket.Write([]byte{data}); err != nil {
			return wrapError(ErrSocket, err)
		}

	case n <= math.MaxUint16:
		data := 0x80 | byte(126)
		if _, err := socket.Write([]byte{data}); err != nil {
			return wrapError(ErrSocket, err)
		}

		var length [2]byte
		binary.BigEndian.PutUint16(length[:], uint16(n))
		if _, err := socket.Write(length[:]); err != nil {
			return wrapError(ErrSocket, err)
		}

	default:
		data := 0x80 | byte(127)
		if _, err := socket.Write([]byte{data}); err != nil {
			return wrapError(ErrSocket, err)
		}

		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(n))
		if _, err := socket.Write(length[:]); err != nil {
			return wrapError(ErrSocket, err)
		}
	}

	// Generate Mask
	var masks [4]byte
	rand.Read(masks[:])
	if _, err := socket.Write(masks[:]); err != nil {
		return wrapError(ErrSocket, err)
	}

	// Mask and send Payload
	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ masks[i%4]
	}
	if _, err := socket.Write(masked); err != nil {
		return wrapError(ErrSocket, err)
	}

	return nil
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
