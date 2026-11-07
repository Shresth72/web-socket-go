package ws

import "fmt"

type ErrorKind struct{ msg string }

var (
	ErrClientHandshake = ErrorKind{"ws: client handshake failed"}
	ErrSocket          = ErrorKind{"ws: socket error"}
	ErrAllocator       = ErrorKind{"ws: allocator error"}
	ErrServerClose     = ErrorKind{"ws: server close error"}
)

func (k ErrorKind) Error() string { return k.msg }

var ErrorKinds = []ErrorKind{
	ErrClientHandshake,
	ErrSocket,
	ErrAllocator,
	ErrServerClose,
}

func wrapError(kind ErrorKind, cause error) error {
	return fmt.Errorf("%w: %w", kind, cause)
}
