// Package finetune builds and validates JSONL datasets for fine-tuning the SDP
// task complexity classifier (F133 dispatch routing).
package finetune

// Message is one chat turn in a fine-tune sample.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Sample is one JSONL line: a chat conversation the model learns to imitate.
type Sample struct {
	Messages []Message `json:"messages"`
	// Meta is stripped before writing to the JSONL but kept in-memory for
	// dedup, splitting, and reporting.
	Meta SampleMeta `json:"-"`
}

// SampleMeta carries provenance for a Sample: where the example came from and
// what the derived label is.
type SampleMeta struct {
	Source   string `json:"source"`    // "ws" | "beads" | "synthetic"
	SourceID string `json:"source_id"` // ws_id, bead id, ...
	Real     bool   `json:"real"`      // true for ws/beads, false for synthetic
	Label    Label  `json:"label"`     // assistant target, parsed
	InputKey string `json:"input_key"` // hash of user content for dedup
}

// Label is the classifier output schema. Assistant content must marshal to
// exactly this struct.
type Label struct {
	Complexity string `json:"complexity"` // low | medium | high
	TaskType   string `json:"task_type"`  // feature | bugfix | refactor | test | docs
	Risk       string `json:"risk"`       // low | high
}

// Split holds the train/eval partition.
type Split struct {
	Train []Sample
	Eval  []Sample
}
