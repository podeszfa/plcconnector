//go:build !windows

package plcconnector

import (
	"fmt"
	"syscall"
)

func sockControl(network, address string, c syscall.RawConn) error {
	return c.Control(func(fd uintptr) {
		err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		if err != nil {
			fmt.Println("SO_REUSEADDR:", err)
			return
		}
	})
}
