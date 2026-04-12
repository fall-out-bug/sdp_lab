//go:build linux

package architect

import (
	"fmt"
	"os"
)

func realPathFromFD(fd int) (string, error) {
	linkPath := fmt.Sprintf("/proc/self/fd/%d", fd)
	return os.Readlink(linkPath)
}
