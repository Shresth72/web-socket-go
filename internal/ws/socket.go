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

	// Send MASK bit and payload length
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

	var key [4]byte
	rand.Read(key[:])
	if _, err := socket.Write(key[:]); err != nil {
		return wrapError(ErrSocket, err)
	}

	// Mask the payload and send it
	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ key[i%4]
	}
	if _, err := socket.Write(masked); err != nil {
		return wrapError(ErrSocket, err)
	}

	return nil
}
