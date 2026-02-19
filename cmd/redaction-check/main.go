package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/redaction"
)

func main() {
	filePath := flag.String("file", "", "Path to candidate OSS export file")
	flag.Parse()
	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "--file is required")
		os.Exit(2)
	}

	b, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read file: %v\n", err)
		os.Exit(1)
	}

	violations := redaction.CheckContent(string(b))
	out := map[string]any{
		"file":       *filePath,
		"ok":         len(violations) == 0,
		"violations": violations,
	}
	jb, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(jb))

	if len(violations) > 0 {
		os.Exit(2)
	}
}
