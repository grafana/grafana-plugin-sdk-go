package testutil

import (
	"fmt"
	"net"
)

// GetFreePort returns a random free port listening on 127.0.0.1.
func GetFreePort() (int, error) {
	a, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("resolve tcp addr: %w", err)
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return 0, fmt.Errorf("listen tcp: %w", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err = l.Close(); err != nil {
		return 0, fmt.Errorf("close: %w", err)
	}
	return port, nil
}
