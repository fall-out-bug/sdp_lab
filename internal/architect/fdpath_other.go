//go:build !linux && !darwin

package architect

import "fmt"

func realPathFromFD(fd int) (string, error) {
	return "", fmt.Errorf("unsupported OS")
}
