package pkg_c

import (
	"github.com/example/circular/pkg_a"
)

func C() {
	// Completing the cycle: C -> A
	pkg_a.A()
}
