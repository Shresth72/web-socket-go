package ws

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

var headerTerminator = []byte("\r\n\r\n")

func (m Message) Payload() []byte {
	var payload []byte
	for _, chunk := range m.Chunks {
		payload = append(payload, chunk.Payload...)
	}
	return payload
}

func isControl(opcode Opcode) bool {
	return 0x8 <= opcode && opcode <= 0xF
}

// base64(SHA1(Sec-WebSocket-Key + GUID)).
func computeSecWebsocketAccept(secWebsocketKey string) string {
	sum := sha1.Sum([]byte(secWebsocketKey + WEB_SOCKET_GUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func validSecWebsocketKey(key string) bool {
	raw, err := base64.StdEncoding.DecodeString(key)
	return err == nil && len(raw) == 16
}

func readHeaders(socket Socket) ([]byte, error) {
	var headers []byte
	var b [1]byte

	for {
		if _, err := io.ReadFull(socket, b[:]); err != nil {
			return nil, err
		}

		headers = append(headers, b[0])
		if bytes.HasSuffix(headers, headerTerminator) {
			return headers, nil
		}

		if len(headers) >= MAX_HEADER_SIZE {
			return nil, fmt.Errorf("ws: header block exceeds %d bytes", MAX_HEADER_SIZE)
		}
	}
}

func headerValue(headers, name string) (string, bool) {
	for _, line := range strings.Split(headers, "\r\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), name) {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}
