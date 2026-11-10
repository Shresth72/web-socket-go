package main

import (
	"errors"
	"io"
	"log"
	"net"
	"os"

	"github.com/Shresth72/ws/internal/ws"
)

const defaultAddr = "127.0.0.1:9000"

func main() {
	addr := defaultAddr
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	log.Printf("listening on ws://%s", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	if err := ws.ServerHandshake(conn); err != nil {
		log.Printf("%s: handshake: %v", conn.RemoteAddr(), err)
		return
	}

	log.Printf("%s: connected", conn.RemoteAddr())

	for {
		msg, err := ws.ReadServerMessage(conn)
		if err != nil {
			switch {
			case errors.Is(err, ws.ErrServerClose):
				log.Printf("%s: closed by client", conn.RemoteAddr())
			case errors.Is(err, io.EOF):
				log.Printf("%s: disconnected", conn.RemoteAddr())
			default:
				log.Printf("%s: read: %v", conn.RemoteAddr(), err)
			}
			return
		}

		payload := msg.Payload()
		log.Printf("%s: received %d bytes", conn.RemoteAddr(), len(payload))

		if err := ws.SendServerMessage(conn, msg.Kind, payload, 1024); err != nil {
			log.Printf("%s: send: %v", conn.RemoteAddr(), err)
			return
		}
	}
}
