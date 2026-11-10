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
func NetDialTCP() {
	conn, err := dial(LOCAL_HOST, LOCAL_SERVICE)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	message := []byte("Hello from the local TCP server!")
	if err := ws.SendClientMessage(conn, ws.MESSAGE_TEXT, message, 1024); err != nil {
		log.Fatalf("send: %v", err)
	}
	fmt.Printf("client: %s\n", message)

	reply, err := ws.ReadClientMessage(conn)
	if err != nil {
		log.Fatalf("read: %v", err)
	}

	fmt.Printf("server: %s\n", reply.Payload())
}
