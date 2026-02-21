// openclaw-agent is the entrypoint for OpenClaw runtime.
// It implements the same protocol as opencode-agent but delegates to OpenClaw.
//
// Usage: openclaw-agent [--loop] [--interval 30s]
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "openclaw-agent: OpenClaw runtime stub; use opencode-agent for current execution")
	os.Exit(1)
}
