package main

import (
	"fmt"
	"log"

	"github.com/Shresth72/ws/internal/ws"
)

const (
	HOST    = "echo.websocket.org"
	SERVICE = "443"
)

func main() {
	conn, err := ws.DialTLS(HOST, SERVICE)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// echo.websocket.org sends a greeting as soon as the connection opens.
	greeting, err := ws.ReadMessage(conn)
	if err != nil {
		log.Fatalf("read greeting: %v", err)
	}
	fmt.Printf("server: %s\n", greeting.Payload())

	message := []byte("Hello from Go WebSocket!")
	if err := ws.SendMessage(conn, ws.MESSAGE_TEXT, message, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}

	reply, err := ws.ReadMessage(conn)
	if err != nil {
		log.Fatalf("read: %v", err)
	}

	fmt.Printf("echo: %s\n", reply.Payload())
}
