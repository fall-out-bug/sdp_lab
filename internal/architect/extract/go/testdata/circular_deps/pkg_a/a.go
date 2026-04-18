package pkg_a

import (
	"github.com/example/circular/pkg_b"
)

func A() {
	// This creates a dependency cycle: A -> B -> C -> A
	pkg_b.B()
}
