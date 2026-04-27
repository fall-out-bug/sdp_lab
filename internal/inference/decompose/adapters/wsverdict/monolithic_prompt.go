package wsverdict

import "fmt"

// monolithicPrompt builds the single combined prompt that asks for a complete
// FinalVerdict JSON in one LLM call. It is the concatenation of the extract +
// classify + aggregate instructions, kept as a straight concat without tuning
// (tuning is bench-driven follow-up if accuracy regresses per 00-146-05).
func monolithicPrompt(d Diff) string {
	return fmt.Sprintf(`You are a senior code reviewer evaluating a workstream merge request.

Analyze the following diff and workstream context, then produce a complete verdict as a JSON object:
{
  "verdict": "passed|partial|failed",
  "score": <float 0.0-1.0>,
  "summary": "<one sentence rationale>",
  "blocking_gates": ["<list of blocking issues, empty for passed>"]
}

Verdict rules:
- "passed": all acceptance criteria met, quality gates pass, code is production-ready
- "partial": most criteria met, minor gaps or concerns remain
- "failed": critical criteria unmet, quality gates failing, or blocking issues present

Workstream: %s
Context: %s

Diff:
%s

Return ONLY the JSON object, no markdown, no explanation.`, d.WSID, d.Context, d.DiffText)
}
