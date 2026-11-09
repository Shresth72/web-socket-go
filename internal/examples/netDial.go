package examples

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Shresth72/ws/internal/ws"
)

const (
	HOST    = "echo.websocket.org"
	SERVICE = "443"

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

	greeting, err := ws.ReadMessage(conn)
	if err != nil {
		log.Fatalf("read greeting: %v", err)
	}
	fmt.Printf("server: %s\n", greeting.Payload())

	message := []byte("Hello from Go Websocket Client!")
	if err := ws.SendMessage(conn, ws.MESSAGE_TEXT, message, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}
	fmt.Printf("client (echo): %s\n", message)

	reply, err := ws.ReadMessage(conn)
	if err != nil {
		log.Fatalf("read: %v", err)
	}

	fmt.Printf("server: %s\n", reply.Payload())
}
