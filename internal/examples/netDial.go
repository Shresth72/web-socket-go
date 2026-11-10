package examples

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Shresth72/ws/internal/ws"
)

const (
	HOST    = "echo.websocket.org"
	SERVICE = "443"

	LOCAL_HOST    = "127.0.0.1"
	LOCAL_SERVICE = "9000"

	DIAL_TIMEOUT = 10 * time.Second
)

func dial(host, service string) (net.Conn, error) {
	addr := net.JoinHostPort(host, service)

	conn, err := net.DialTimeout("tcp", addr, DIAL_TIMEOUT)
	if err != nil {
		return nil, err
	}

	if err := ws.ClientHandshake(conn, host); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func dialTLS(host, service string) (net.Conn, error) {
	addr := net.JoinHostPort(host, service)
	dialier := net.Dialer{Timeout: DIAL_TIMEOUT}

	conn, err := tls.DialWithDialer(&dialier, "tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return nil, err
	}

	if err := ws.ClientHandshake(conn, host); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func NetDialEcho() {
	conn, err := dialTLS(HOST, SERVICE)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	greeting, err := ws.ReadClientMessage(conn)
	if err != nil {
		log.Fatalf("read greeting: %v", err)
	}
	fmt.Printf("server: %s\n", greeting.Payload())

	message := []byte("Hello from Go Websocket Client!")
	if err := ws.SendClientMessage(conn, ws.MESSAGE_TEXT, message, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}
	fmt.Printf("client (echo): %s\n", message)

	reply, err := ws.ReadClientMessage(conn)
	if err != nil {
		log.Fatalf("read: %v", err)
	}

	fmt.Printf("server: %s\n", reply.Payload())
}

// NetDialTCP talks to the local server in cmd/server over plain TCP.
// It sends a message, reads the echo, then sends PING and CLOSE frames.
func NetDialTCP() {
	conn, err := dial(LOCAL_HOST, LOCAL_SERVICE)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// 1
	message1 := []byte("Hello from the local TCP server!")
	if err := ws.SendClientMessage(conn, ws.MESSAGE_TEXT, message1, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}
	fmt.Printf("client: %s\n", message1)

	reply1, err := ws.ReadClientMessage(conn)
	if err != nil {
		log.Fatalf("read: %v", err)
	}
	fmt.Printf("server: %s\n", reply1.Payload())

	// Ping waits for the server's PONG instead of firing and forgetting.
	pingPayload := []byte("ping")
	pong, err := ws.SendPing(conn, pingPayload, true)
	if err != nil {
		log.Fatalf("ping: %v", err)
	}
	fmt.Printf("client: PING %q\n", pingPayload)
	fmt.Printf("server: PONG %q\n", pong)

	// 2
	message2 := []byte("Hello from the local TCP server 2!")
	if err := ws.SendClientMessage(conn, ws.MESSAGE_TEXT, message2, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}
	fmt.Printf("client: %s\n", message2)

	reply2, err := ws.ReadClientMessage(conn)
	if err != nil {
		log.Fatalf("read: %v", err)
	}
	fmt.Printf("server: %s\n", reply2.Payload())

	if err := ws.SendClose(conn, []byte{0x03, 0xE8}, true); err != nil { // 1000
		log.Fatalf("close: %v", err)
	}
	fmt.Println("client: CLOSE 1000")

	// Skips the PONG, then returns ErrServerClose on the server's CLOSE reply.
	if _, err := ws.ReadClientMessage(conn); !errors.Is(err, ws.ErrServerClose) {
		log.Fatalf("expected CLOSE reply, got %v", err)
	}
	fmt.Println("server: CLOSE")
}
