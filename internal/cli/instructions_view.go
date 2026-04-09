package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

const instructionSpecVersion = "v1.0"

type InstructionPayload struct {
	SpecVersion  string        `json:"spec_version"`
	Context      string        `json:"context,omitempty"`
	Instructions []Instruction `json:"instructions"`
}

type Instruction struct {
	Step            int    `json:"step"`
	Action          string `json:"action"`
	Reason          string `json:"reason"`
	Command         string `json:"command,omitempty"`
	ExpectedOutcome string `json:"expected_outcome,omitempty"`
	Troubleshooting string `json:"troubleshooting,omitempty"`
}

func NewInstructionPayloadForAction(action string, status *StatusView) *InstructionPayload {
	payload := &InstructionPayload{
		SpecVersion: instructionSpecVersion,
		Context:     fmt.Sprintf("Action: %s", action),
	}

	switch action {
	case "continue":
		payload.Instructions = instructionsForContinue(status)
	case "start":
		payload.Instructions = instructionsForStart(status)
	case "resolve_blockers":
		payload.Instructions = instructionsForResolveBlockers(status)
	case "check_status":
		payload.Instructions = instructionsForCheckStatus(status)
	default:
		payload.Instructions = instructionsForDefault(status)
	}

	return payload
}

func (p *InstructionPayload) RenderText() string {
	var b strings.Builder

	if p.Context != "" {
		b.WriteString("Context: " + p.Context + "\n\n")
	}

	b.WriteString("Instructions:\n")
	for _, instr := range p.Instructions {
		fmt.Fprintf(&b, "\n%d. %s\n", instr.Step, instr.Action)
		fmt.Fprintf(&b, "   Reason: %s\n", instr.Reason)
		if instr.Command != "" {
			fmt.Fprintf(&b, "   Command: %s\n", instr.Command)
		}
		if instr.ExpectedOutcome != "" {
			fmt.Fprintf(&b, "   Expected: %s\n", instr.ExpectedOutcome)
		}
		if instr.Troubleshooting != "" {
			fmt.Fprintf(&b, "   Troubleshooting: %s\n", instr.Troubleshooting)
		}
	}

	return strings.TrimSpace(b.String())
}

func (p *InstructionPayload) RenderJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func instructionsForContinue(status *StatusView) []Instruction {
	if status.InProgressCount == 0 {
		return []Instruction{{
			Step:            1,
			Action:          "No work in progress",
			Reason:          "There is no active work to continue",
			Command:         "sdp-ready",
			ExpectedOutcome: "Shows available ready work",
		}}
	}

	item := status.Items[0]
	return []Instruction{
		{
			Step:            1,
			Action:          fmt.Sprintf("Review current work on %s", item.ID),
			Reason:          "Understand the current state before continuing",
			Command:         fmt.Sprintf("bd show %s", item.ID),
			ExpectedOutcome: "Details of the in-progress issue",
		},
		{
			Step:            2,
			Action:          "Check for blocking issues",
			Reason:          "Ensure no blockers prevent progress",
			Command:         "bd blocked",
			ExpectedOutcome: "List of any blocking issues",
		},
		{
			Step:            3,
			Action:          "Complete the work and close the issue",
			Reason:          "Finish the current task before starting new work",
			Command:         fmt.Sprintf("bd close %s -r \"Work completed\"", item.ID),
			ExpectedOutcome: "Issue closed and synced to git",
		},
	}
}

func instructionsForStart(status *StatusView) []Instruction {
	if status.ReadyCount == 0 {
		return []Instruction{{
			Step:            1,
			Action:          "No ready work available",
			Reason:          "All work is either in progress or blocked",
			Command:         "sdp-ready",
			ExpectedOutcome: "Re-check status for changes",
		}}
	}

	var readyItem *StatusItem
	for i := range status.Items {
		if status.Items[i].Status == "ready" {
			readyItem = &status.Items[i]
			break
		}
	}

	if readyItem == nil {
		return []Instruction{{
			Step:    1,
			Action:  "No ready work found",
			Reason:  "Status inconsistency detected",
			Command: "bd ready",
		}}
	}

	return []Instruction{
		{
			Step:            1,
			Action:          fmt.Sprintf("Claim the issue %s", readyItem.ID),
			Reason:          fmt.Sprintf("%s has the highest priority among ready issues", readyItem.ID),
			Command:         fmt.Sprintf("bd update %s --status in_progress", readyItem.ID),
			ExpectedOutcome: "Issue marked as in_progress",
		},
		{
			Step:            2,
			Action:          "Review the workstream file",
			Reason:          "Understand acceptance criteria and scope",
			Command:         fmt.Sprintf("cat docs/workstreams/backlog/00-%s.md", readyItem.ID),
			ExpectedOutcome: "Workstream details displayed",
		},
		{
			Step:            3,
			Action:          "Create a feature branch",
			Reason:          "Isolate work from main/dev branch",
			Command:         fmt.Sprintf("git checkout -b feature/%s-short-name", readyItem.ID),
			ExpectedOutcome: "New branch created",
		},
	}
}

func instructionsForResolveBlockers(status *StatusView) []Instruction {
	if status.BlockedCount == 0 {
		return []Instruction{{
			Step:            1,
			Action:          "No blockers to resolve",
			Reason:          "All work is either ready or in progress",
			Command:         "sdp-ready",
			ExpectedOutcome: "Shows available work",
		}}
	}

	blockers := status.NextAction.BlockingIssues
	instructions := []Instruction{
		{
			Step:            1,
			Action:          "Review blocking issues",
			Reason:          fmt.Sprintf("%d issue(s) are blocked - identify root causes", status.BlockedCount),
			Command:         "bd blocked",
			ExpectedOutcome: "List of blocked issues and their blockers",
		},
	}

	for i, blocker := range blockers {
		instructions = append(instructions, Instruction{
			Step:            i + 2,
			Action:          fmt.Sprintf("Resolve blocker: %s", blocker),
			Reason:          "This issue blocks other work from proceeding",
			Command:         fmt.Sprintf("bd show %s", blocker),
			ExpectedOutcome: "Blocker details and path to resolution",
		})
	}

	instructions = append(instructions, Instruction{
		Step:            len(instructions) + 1,
		Action:          "Re-check ready status after resolving blockers",
		Reason:          "Blocked issues should become ready once blockers are cleared",
		Command:         "sdp-ready",
		ExpectedOutcome: "Updated status with newly ready work",
	})

	return instructions
}

func instructionsForCheckStatus(status *StatusView) []Instruction {
	return []Instruction{
		{
			Step:            1,
			Action:          "View current status",
			Reason:          "Get a complete picture of ready, blocked, and in-progress work",
			Command:         "sdp-ready",
			ExpectedOutcome: "Status summary with next action recommendation",
		},
		{
			Step:            2,
			Action:          "View JSON output for programmatic access",
			Reason:          "Machine-readable format for automation and scripts",
			Command:         "sdp-ready --format json",
			ExpectedOutcome: "JSON status conforming to StatusView contract",
		},
		{
			Step:            3,
			Action:          "Get detailed instructions for next action",
			Reason:          "Step-by-step guidance for the recommended action",
			Command:         fmt.Sprintf("sdp-ready --instructions --action %s", status.NextAction.Recommended),
			ExpectedOutcome: "InstructionPayload with detailed steps",
		},
	}
}

func instructionsForDefault(status *StatusView) []Instruction {
	return []Instruction{
		{
			Step:            1,
			Action:          "Check current status",
			Reason:          "Understand the current state of work",
			Command:         "sdp-ready",
			ExpectedOutcome: "Status summary with next action",
		},
		{
			Step:    2,
			Action:  fmt.Sprintf("Follow recommendation: %s", status.NextAction.Recommended),
			Reason:  status.NextAction.Reason,
			Command: status.NextAction.Command,
		},
	}
}
