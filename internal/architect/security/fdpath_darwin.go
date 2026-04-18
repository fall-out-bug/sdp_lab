//go:build darwin

package security

import (
	"fmt"
	"syscall"
	"unsafe"
)

// realPathFromFD gets the real path from a file descriptor on Darwin/macOS.
// It uses fcntl(fd, F_GETPATH, ...) which is the Darwin-specific way to get
// the real path of an open file descriptor.
func realPathFromFD(fd int) (string, error) {
	const PATH_MAX = 1024 // PATH_MAX on Darwin

	var buf [PATH_MAX]byte

	// F_GETPATH is the Darwin-specific fcntl command to get the path
	// from a file descriptor
	_, _, errno := syscall.Syscall(
		syscall.SYS_FCNTL,
		uintptr(fd),
		uintptr(syscall.F_GETPATH),
		uintptr(unsafe.Pointer(&buf[0])),
	)

	if errno != 0 {
		return "", fmt.Errorf("fcntl F_GETPATH failed: %w", errno)
	}

	// Find the null terminator
	length := 0
	for i, b := range buf {
		if b == 0 {
			length = i
			break
		}
		// Safety check - if we didn't find null terminator within PATH_MAX
		if i == PATH_MAX-1 {
			return "", fmt.Errorf("path not null-terminated within PATH_MAX")
		}
	}

	return string(buf[:length]), nil
}
