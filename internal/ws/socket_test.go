package ws

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
)

type hexSocket struct{}

func (hexSocket) Read(p []byte) (int, error) { return os.Stdin.Read(p) }

func (hexSocket) Write(p []byte) (int, error) {
	const max = 64
	if len(p) > max {
		fmt.Printf("write %d bytes (first %d):\n%s...\n", len(p), max, hex.Dump(p[:max]))
	} else {
		fmt.Printf("write %d bytes:\n%s", len(p), hex.Dump(p))
	}
	return len(p), nil
}

func TestSendFrameSmall(t *testing.T) {
	if err := sendFrame(hexSocket{}, true, OPCODE_TEXT, []byte("hello")); err != nil {
		t.Fatal(err)
	}
}

func TestSendFrameMedium(t *testing.T) {
	if err := sendFrame(hexSocket{}, true, OPCODE_BIN, bytes.Repeat([]byte("a"), 300)); err != nil {
		t.Fatal(err)
	}
}

func TestSendFrameLarge(t *testing.T) {
	if err := sendFrame(
		hexSocket{},
		true,
		OPCODE_BIN,
		bytes.Repeat([]byte("b"), 70000),
	); err != nil {
		t.Fatal(err)
	}
}
