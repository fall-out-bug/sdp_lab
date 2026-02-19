package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/runtimeparity"
)

func load(path string) (runtimeparity.CapabilitySet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return runtimeparity.CapabilitySet{}, err
	}
	var out runtimeparity.CapabilitySet
	if err := json.Unmarshal(b, &out); err != nil {
		return runtimeparity.CapabilitySet{}, err
	}
	return out, nil
}

func main() {
	aPath := flag.String("a", "", "First runtime capabilities JSON")
	bPath := flag.String("b", "", "Second runtime capabilities JSON")
	flag.Parse()
	if *aPath == "" || *bPath == "" {
		fmt.Fprintln(os.Stderr, "--a and --b are required")
		os.Exit(2)
	}

	a, err := load(*aPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load a: %v\n", err)
		os.Exit(1)
	}
	b, err := load(*bPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load b: %v\n", err)
		os.Exit(1)
	}

	cmp := runtimeparity.Compare(a, b)
	out := map[string]any{
		"runtime_a":   a.Runtime,
		"runtime_b":   b.Runtime,
		"equal":       cmp.Equal,
		"differences": cmp.Differences,
	}
	jb, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(jb))
	if !cmp.Equal {
		os.Exit(2)
	}
}
