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

	c := ws.NewClient(conn)

	greeting, err := c.ReadMessage()
	if err != nil {
		log.Fatalf("read greeting: %v", err)
	}
	fmt.Printf("server: %s\n", greeting.Payload())

	message := []byte("Hello from Go Websocket Client!")
	if err := c.SendMessage(ws.MESSAGE_TEXT, message, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}
	fmt.Printf("client (echo): %s\n", message)

	reply, err := c.ReadMessage()
	if err != nil {
		log.Fatalf("read: %v", err)
	}

	fmt.Printf("server: %s\n", reply.Payload())
}

func NetDialTCP() {
	conn, err := dial(LOCAL_HOST, LOCAL_SERVICE)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	c := ws.NewClient(conn)

	// 1
	message1 := []byte("Hello from the local TCP server!")
	if err := c.SendMessage(ws.MESSAGE_TEXT, message1, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}
	fmt.Printf("client: %s\n", message1)

	reply1, err := c.ReadMessage()
	if err != nil {
		log.Fatalf("read: %v", err)
	}
	fmt.Printf("server: %s\n", reply1.Payload())

	// 2
	pingPayload := []byte("ping")
	if err := c.SendPing(pingPayload); err != nil {
		log.Fatalf("ping: %v", err)
	}
	fmt.Printf("client: PING %q\n", pingPayload)
	fmt.Printf("server: PONG %q\n", pingPayload) // RFC 6455: PONG echoes the PING payload

	// 3
	message2 := []byte("Hello from the local TCP server 2!")
	if err := c.SendMessage(ws.MESSAGE_TEXT, message2, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}
	fmt.Printf("client: %s\n", message2)

	reply2, err := c.ReadMessage()
	if err != nil {
		log.Fatalf("read: %v", err)
	}
	fmt.Printf("server: %s\n", reply2.Payload())

	// 4
	if err := c.SendClose([]byte{0x03, 0xE8}); err != nil { // 1000
		log.Fatalf("close: %v", err)
	}
	fmt.Println("client: CLOSE 1000")

	if _, err := c.ReadMessage(); !errors.Is(err, ws.ErrServerClose) {
		log.Fatalf("expected CLOSE reply, got %v", err)
	}
	fmt.Println("server: CLOSE")
}
