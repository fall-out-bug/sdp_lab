package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func runTrace(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp trace <card-id>")
		os.Exit(2)
	}
	cardID := args[0]
	store := openStore()
	projectRoot := store.ProjectRoot

	card, err := store.LoadCardByID(cardID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load card: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📌 %s: %s\n", card.ID, card.Title)
	fmt.Printf("   Status: %s | Phase: %s | Executor: %s\n", card.Status, card.TaskType, card.ExecutorRuntimeState)
	if card.NormalizedIntent != "" {
		fmt.Printf("   Intent: %s\n", card.NormalizedIntent)
	}

	artDir := filepath.Join(projectRoot, ".sdp", "artifacts", cardID)
	entries, err := os.ReadDir(artDir)
	if err != nil || len(entries) == 0 {
		fmt.Println("   (no evidence)")
		return
	}

	phaseOrder := map[string]int{"clarification": 0, "plan": 1, "build": 2, "evaluation": 3, "deploy-staging": 4, "deploy": 5, "summary": 6}
	type evFile struct{ name, phase, ts string; data map[string]any }
	var files []evFile

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fpath := filepath.Join(artDir, e.Name())
		data, readErr := os.ReadFile(fpath)
		if readErr != nil {
			continue
		}
		var parsed map[string]any
		if filepath.Ext(e.Name()) == ".json" {
			_ = json.Unmarshal(data, &parsed)
		}
		phase := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		ts := ""
		if parsed != nil {
			if t, ok := parsed["timestamp"].(string); ok {
				ts = t
			}
		}
		if ts == "" {
			info, _ := e.Info()
			ts = info.ModTime().UTC().Format("15:04:05")
		}
		files = append(files, evFile{name: e.Name(), phase: phase, ts: ts, data: parsed})
	}

	slices.SortFunc(files, func(a, b evFile) int {
		pa, pb := phaseOrder[a.phase], phaseOrder[b.phase]
		if c := cmp.Compare(pa, pb); c != 0 {
			return c
		}
		return cmp.Compare(a.ts, b.ts)
	})

	fmt.Println()
	for _, f := range files {
		switch f.phase {
		case "clarification":
			status := "—"
			if f.data != nil {
				if s, ok := f.data["status"].(string); ok {
					status = s
				}
			}
			fmt.Printf("   🔍 Clarification [%s] %s\n", status, f.ts)
			if f.data != nil {
				if qs, ok := f.data["questions"].([]any); ok {
					for _, q := range qs {
						if s, ok := q.(string); ok && s != "" {
							fmt.Printf("      ❓ %s\n", s)
						}
					}
				}
			}
		case "build":
			exitCode := "?"
			executorName := "?"
			if f.data != nil {
				if ec, ok := f.data["exit_code"].(float64); ok {
					exitCode = fmt.Sprintf("%d", int(ec))
				}
				if ex, ok := f.data["executor"].(string); ok {
					executorName = ex
				}
			}
			mark := "✅"
			if exitCode != "0" {
				mark = "❌"
			}
			fmt.Printf("   🔨 Build %s [exit=%s] %s | agent=%s\n", mark, exitCode, f.ts, executorName)
			if f.data != nil {
				if arts, ok := f.data["artifacts"].([]any); ok {
					for _, a := range arts {
						if m, ok := a.(map[string]any); ok {
							if desc, ok := m["description"].(string); ok {
								fmt.Printf("      → %s\n", desc)
							} else if ref, ok := m["reference"].(string); ok {
								fmt.Printf("      → %s\n", ref)
							}
						}
					}
				}
			}
		case "evaluation":
			verdict := "?"
			score := "?"
			if f.data != nil {
				if v, ok := f.data["verdict"].(string); ok {
					verdict = v
				}
				if s, ok := f.data["score"].(float64); ok {
					score = fmt.Sprintf("%.0f%%", s*100)
				}
			}
			mark := "✅"
			if verdict != "pass" {
				mark = "❌"
			}
			fmt.Printf("   📋 Evaluation %s [score=%s] %s\n", mark, score, f.ts)
			if f.data != nil {
				if passed, ok := f.data["passed"].(map[string]any); ok {
					for crit, val := range passed {
						icon := "✅"
						if v, ok := val.(bool); ok && !v {
							icon = "❌"
						}
						fmt.Printf("      %s %s\n", icon, crit)
					}
				}
				if findings, ok := f.data["findings"].([]any); ok {
					for _, f2 := range findings {
						if s, ok := f2.(string); ok {
							fmt.Printf("      ⚠️  %s\n", s)
						}
					}
				}
			}
		case "summary":
			if f.data != nil {
				if text, ok := f.data["text"].(string); ok {
					fmt.Printf("   📝 Summary %s\n      %s\n", f.ts, text)
				}
			}
		default:
			fmt.Printf("   📄 %s %s\n", f.name, f.ts)
		}
	}
}
