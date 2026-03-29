package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	// Shift so each subcommand's FlagSet sees os.Args[1:] as its own args.
	os.Args = os.Args[1:]

	var err error
	switch cmd {
	case "route":
		err = runRoute()
	case "limits":
		err = runLimits()
	case "profile":
		err = runProfile()
	case "bench":
		err = runBench()
	case "compare":
		err = runCompare()
	case "status":
		err = runStatus()
	case "help", "--help", "-h":
		usage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n", cmd)
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sdp-dispatch <command> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  route    Select the best harness/model for a task")
	fmt.Fprintln(os.Stderr, "  limits   Check provider rate-limit availability")
	fmt.Fprintln(os.Stderr, "  profile  List capability profiles")
	fmt.Fprintln(os.Stderr, "  bench    Run benchmarks for harnesses against a task")
	fmt.Fprintln(os.Stderr, "  compare  Compare harnesses for a specific task")
	fmt.Fprintln(os.Stderr, "  status   Show current dispatch state")
	fmt.Fprintln(os.Stderr, "  help     Show this help message")
}
