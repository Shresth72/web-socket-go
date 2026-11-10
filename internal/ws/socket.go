package ws

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type Socket interface {
	io.Reader
	io.Writer
}

// sendFrame writes a single frame.
// Client-to-Server frames must be masked;
// Server-to-Client frames must not be masked (RFC 6455 section 5.1).
func sendFrame(socket Socket, fin bool, opcode Opcode, payload []byte, mask bool) error {
	if isControl(opcode) {
		if !fin {
			return fmt.Errorf("ws: control frame must not be fragmented")
		}
		if len(payload) > 125 {
			return fmt.Errorf("ws: control frame payload too large: %d", len(payload))
		}
	}

	// Send FIN and OPCODE
	data := opcode & 0x0F
	if fin {
		data |= 0x80 // 1<<7
	}

	if _, err := socket.Write([]byte{data}); err != nil {
		return wrapError(ErrSocket, err)
	}

	// Send MASK bit and Payload length
	maskBit := byte(0)
	if mask {
		maskBit = 0x80
	}

	n := len(payload)
	switch {
	case n <= 125:
		if _, err := socket.Write([]byte{maskBit | byte(n)}); err != nil {
			return wrapError(ErrSocket, err)
		}

	case n <= math.MaxUint16:
		if _, err := socket.Write([]byte{maskBit | 126}); err != nil {
			return wrapError(ErrSocket, err)
		}

		var length [2]byte
		binary.BigEndian.PutUint16(length[:], uint16(n))
		if _, err := socket.Write(length[:]); err != nil {
			return wrapError(ErrSocket, err)
		}

	default:
		if _, err := socket.Write([]byte{maskBit | 127}); err != nil {
			return wrapError(ErrSocket, err)
		}

		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(n))
		if _, err := socket.Write(length[:]); err != nil {
			return wrapError(ErrSocket, err)
		}
	}

	if !mask {
		if _, err := socket.Write(payload); err != nil {
			return wrapError(ErrSocket, err)
		}
		return nil
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

func readFrame(socket Socket) (Frame, error) {
	var header [2]byte
	if _, err := io.ReadFull(socket, header[:]); err != nil {
		return Frame{}, wrapError(ErrSocket, err)
	}

	fin := header[0]&0x80 != 0
	opcode := header[0] & 0x0F
	lengthCode := header[1] & 0x7F
	masked := header[1]&0x80 != 0

	var payloadLength uint64
	switch lengthCode {
	case 126:
		var extendedLength [2]byte
		if _, err := io.ReadFull(socket, extendedLength[:]); err != nil {
			return Frame{}, wrapError(ErrSocket, err)
		}
		payloadLength = uint64(binary.BigEndian.Uint16(extendedLength[:]))

	case 127:
		var extendedLength [8]byte
		if _, err := io.ReadFull(socket, extendedLength[:]); err != nil {
			return Frame{}, wrapError(ErrSocket, err)
		}
		payloadLength = binary.BigEndian.Uint64(extendedLength[:])

		if payloadLength>>63 != 0 {
			return Frame{}, fmt.Errorf("ws: invalid payload length: %d", payloadLength)
		}

	default:
		payloadLength = uint64(lengthCode)
	}

	if isControl(opcode) {
		if !fin {
			return Frame{}, fmt.Errorf("ws: control frame must not be fragmented")
		}
		if payloadLength > 125 {
			return Frame{}, fmt.Errorf("ws: control frame payload too large: %d", payloadLength)
		}
	}

	const maxInt = int(^uint(0) >> 1)
	if payloadLength > uint64(maxInt) {
		return Frame{}, fmt.Errorf("ws: payload too large: %d", payloadLength)
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(socket, maskKey[:]); err != nil {
			return Frame{}, wrapError(ErrSocket, err)
		}
	}

	payload := make([]byte, int(payloadLength))
	if _, err := io.ReadFull(socket, payload); err != nil {
		return Frame{}, wrapError(ErrSocket, err)
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return Frame{
		Fin:     fin,
		Opcode:  opcode,
		Payload: payload,
	}, nil
}

func SendPing(socket Socket, payload []byte, mask bool) error {
	return sendFrame(socket, true, OPCODE_PING, payload, mask)
}

func SendPong(socket Socket, payload []byte, mask bool) error {
	return sendFrame(socket, true, OPCODE_PONG, payload, mask)
}

func SendClose(socket Socket, payload []byte, mask bool) error {
	if len(payload) == 1 {
		return fmt.Errorf("ws: close payload must be empty or at least 2 bytes")
	}
	return sendFrame(socket, true, OPCODE_CLOSE, payload, mask)
}
