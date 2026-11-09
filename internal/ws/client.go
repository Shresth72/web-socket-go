package ws

import (
	"crypto/tls"
	"net"
	"time"
)

const dialTimeout = 10 * time.Second

func Dial(host, service string) (net.Conn, error) {
	addr := net.JoinHostPort(host, service)

	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, wrapError(ErrSocket, err)
	}

	if err := clientHandshake(conn, []byte(host)); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func DialTLS(host, service string) (net.Conn, error) {
	addr := net.JoinHostPort(host, service)
	dialer := &net.Dialer{Timeout: dialTimeout}

	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName: host,
	})
	if err != nil {
		return nil, wrapError(ErrSocket, err)
	}

	if err := clientHandshake(conn, []byte(host)); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}
