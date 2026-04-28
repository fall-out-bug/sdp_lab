package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/control"
	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
)

func openStore() *control.Store {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get cwd: %v\n", err)
		os.Exit(1)
	}
	root, err := orchestrate.FindProjectRoot(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: find project root: %v\n", err)
		os.Exit(1)
	}
	store, err := control.OpenFromEnv(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open control store: %v\n", err)
		os.Exit(1)
	}
	return store
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func flagSetFrom(args []string, flag string) *bool {
	v := false
	for _, a := range args {
		if a == flag || a == flag+"=true" {
			v = true
		}
	}
	return &v
}
