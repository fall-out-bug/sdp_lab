package main

import (
	"fmt"
	"os"

	"sdp_dev/internal/cli"
)

func runDoctor(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp doctor control")
		os.Exit(2)
	}
	switch args[0] {
	case "control":
		runDoctorControl()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown doctor subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func runDoctorControl() {
	store := openStore()
	report, err := store.DoctorControl()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: doctor control: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(cli.RenderDoctorControl(report))
	if len(report.Checks) > 0 {
		os.Exit(1)
	}
}
