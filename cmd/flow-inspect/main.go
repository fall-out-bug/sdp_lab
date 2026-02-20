package main

import (
  "encoding/json"
  "flag"
  "fmt"
  "os"
  "path/filepath"
)

func main() {
  issueID := flag.String("issue", "", "Issue ID")
  flag.Parse()
  if *issueID == "" {
    fmt.Fprintln(os.Stderr, "--issue is required")
    os.Exit(2)
  }
  wd, _ := os.Getwd()
  runPath := filepath.Join(wd, ".sdp", "runs", *issueID+".json")
  payload, err := os.ReadFile(runPath)
  if err != nil {
    fmt.Fprintf(os.Stderr, "read run packet: %v\n", err)
    os.Exit(1)
  }
  var run map[string]any
  if err := json.Unmarshal(payload, &run); err != nil {
    fmt.Fprintf(os.Stderr, "parse run packet: %v\n", err)
    os.Exit(1)
  }
  flow, _ := run["flow"].(string)
  if flow == "" {
    flow, _ = run["status"].(string)
  }
  out := map[string]any{
    "issue": *issueID,
    "flow": flow,
    "run": run,
  }
  b, _ := json.MarshalIndent(out, "", "  ")
  fmt.Println(string(b))
}
