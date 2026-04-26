// Command sdp-ft-validate verifies a fine-tune JSONL file against the SDP
// classifier schema. Exits non-zero on any failure.
package main

import (
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/finetune"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		args = []string{
			"internal/dispatch/training/train.jsonl",
			"internal/dispatch/training/eval.jsonl",
		}
	}

	failed := 0
	for _, path := range args {
		res, err := finetune.ValidJSONL(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			failed++
			continue
		}
		fmt.Printf("%s: %d lines, %d unique users, %d errors\n",
			path, res.LineCount, res.UniqueUsers, len(res.Errors))
		for _, e := range res.Errors {
			fmt.Printf("  line %d: %s\n", e.Line, e.Reason)
		}
		if len(res.Errors) > 0 {
			failed++
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
}
