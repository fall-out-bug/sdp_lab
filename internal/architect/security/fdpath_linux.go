//go:build linux

package security

import (
	"fmt"
	"syscall"
)

// realPathFromFD gets the real path from a file descriptor on Linux.
// It reads /proc/self/fd/{fd} which is the standard Linux way to get
// the real path of an open file descriptor.
func realPathFromFD(fd int) (string, error) {
	// Read the symbolic link /proc/self/fd/{fd}
	// The link target is the real path of the file descriptor
	path := fmt.Sprintf("/proc/self/fd/%d", fd)

	// Use readlinkat with AT_FDCWD to read the symlink
	var buf [4096]byte // PATH_MAX on Linux is 4096

	n, err := syscall.Readlink(path, buf[:])
	if err != nil {
		return "", fmt.Errorf("readlink %s failed: %w", path, err)
	}

	if n >= len(buf) {
		return "", fmt.Errorf("path too long (>= %d)", len(buf))
	}

	return string(buf[:n]), nil
}
