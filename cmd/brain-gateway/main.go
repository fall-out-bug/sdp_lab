package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/policy"
)

func main() {
	requestPath := flag.String("request", "", "Path to JSON request payload")
	flag.Parse()
	if *requestPath == "" {
		fmt.Fprintln(os.Stderr, "--request is required")
		os.Exit(2)
	}

	data, err := os.ReadFile(*requestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read request: %v\n", err)
		os.Exit(1)
	}
	var req policy.DecisionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Fprintf(os.Stderr, "parse request: %v\n", err)
		os.Exit(1)
	}

	res := policy.Decide(req)
	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
