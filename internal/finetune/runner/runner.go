// Package runner abstracts a fine-tune backend behind a small interface so the
// SDP CLI can target either the OpenAI fine-tune API or a local MLX run with
// the same control flow (upload → create job → poll status).
package runner

import "context"

// Status is the normalised job state across backends.
type Status string

const (
	StatusUnknown   Status = "unknown"
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// JobInfo describes the state of one fine-tune run.
type JobInfo struct {
	ID         string `json:"id"`
	Status     Status `json:"status"`
	BaseModel  string `json:"base_model"`
	OutputName string `json:"output_name,omitempty"` // path or model tag of the result
	Error      string `json:"error,omitempty"`
	Logs       string `json:"logs,omitempty"`
}

// FileRef points at an uploaded training file. For OpenAI it is a file_id,
// for MLX it is a local path.
type FileRef struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// CreateJobOpts groups the optional knobs every backend can honour.
type CreateJobOpts struct {
	BaseModel string
	Suffix    string  // tag to identify the fine-tune
	Epochs    int     // 0 = backend default
	LR        float64 // 0 = backend default
}

// Runner is the contract every fine-tune backend implements.
type Runner interface {
	Name() string
	Upload(ctx context.Context, jsonlPath string) (FileRef, error)
	CreateJob(ctx context.Context, file FileRef, opts CreateJobOpts) (JobInfo, error)
	Poll(ctx context.Context, jobID string) (JobInfo, error)
}
