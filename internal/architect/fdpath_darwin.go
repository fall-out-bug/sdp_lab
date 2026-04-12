//go:build darwin

package architect

import (
	"bytes"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

func realPathFromFD(fd int) (string, error) {
	buf := make([]byte, 1024)
	_, _, errno := unix.Syscall(unix.SYS_FCNTL, uintptr(fd), unix.F_GETPATH, uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return "", fmt.Errorf("fcntl F_GETPATH: %w", errno)
	}
	n := bytes.IndexByte(buf, 0)
	if n < 0 {
		n = len(buf)
	}
	return string(buf[:n]), nil
}
