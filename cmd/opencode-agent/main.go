package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "SDP_RUNTIME=opencode", "SDP_MODEL=glm-5")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func runCycle() error {
	if out, err := run("go", "run", "./cmd/swarm-worker"); err != nil {
		return err
	} else {
		fmt.Print(string(out))
	}

	if out, err := run("go", "run", "./cmd/swarm-reviewer"); err != nil {
		return err
	} else {
		fmt.Print(string(out))
	}

	return nil
}

func main() {
	loop := flag.Bool("loop", false, "Run continuously")
	interval := flag.Duration("interval", 30*time.Second, "Loop interval")
	flag.Parse()

	if !*loop {
		if err := runCycle(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	for {
		if err := runCycle(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		time.Sleep(*interval)
	}
}
