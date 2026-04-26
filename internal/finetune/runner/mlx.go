package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// MLXRunner shells out to mlx_lm.lora (Apple Silicon LoRA fine-tune CLI).
// "Upload" is a local copy into a workspace directory; "CreateJob" spawns
// mlx_lm.lora and writes the job state to a sidecar JSON; "Poll" reads it.
type MLXRunner struct {
	// Workspace holds per-job dirs (data, logs, adapter, status.json).
	Workspace string
	// LoRACommand is invoked as: <cmd> --train --data <dir> --model <base> --adapter-path <out>
	// Default: mlx_lm.lora
	LoRACommand string
}

// NewMLXRunner uses ~/.sdp/finetune by default.
func NewMLXRunner(workspace string) *MLXRunner {
	if workspace == "" {
		home, _ := os.UserHomeDir()
		workspace = filepath.Join(home, ".sdp", "finetune", "mlx")
	}
	return &MLXRunner{Workspace: workspace, LoRACommand: "mlx_lm.lora"}
}

func (r *MLXRunner) Name() string { return "mlx" }

// Upload copies the JSONL into workspace/<job>/data/train.jsonl. Returns a
// FileRef whose ID is the random job slug — Upload effectively allocates the
// job dir; CreateJob reuses the same slug carried in FileRef.ID.
func (r *MLXRunner) Upload(_ context.Context, jsonlPath string) (FileRef, error) {
	slug, err := randomSlug()
	if err != nil {
		return FileRef{}, err
	}
	jobDir := filepath.Join(r.Workspace, slug)
	dataDir := filepath.Join(jobDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return FileRef{}, fmt.Errorf("mlx: mkdir data: %w", err)
	}
	dst := filepath.Join(dataDir, "train.jsonl")
	if err := copyFile(jsonlPath, dst); err != nil {
		return FileRef{}, err
	}
	return FileRef{ID: slug, Path: dst}, nil
}

// CreateJob starts mlx_lm.lora as a detached subprocess.
func (r *MLXRunner) CreateJob(_ context.Context, file FileRef, opts CreateJobOpts) (JobInfo, error) {
	if opts.BaseModel == "" {
		opts.BaseModel = "mlx-community/Qwen2.5-3B-Instruct-4bit"
	}
	jobDir := filepath.Join(r.Workspace, file.ID)
	dataDir := filepath.Join(jobDir, "data")
	adapterDir := filepath.Join(jobDir, "adapter")
	logFile := filepath.Join(jobDir, "run.log")
	statusFile := filepath.Join(jobDir, "status.json")

	if err := os.MkdirAll(adapterDir, 0o755); err != nil {
		return JobInfo{}, fmt.Errorf("mlx: mkdir adapter: %w", err)
	}

	args := []string{
		"--train",
		"--data", dataDir,
		"--model", opts.BaseModel,
		"--adapter-path", adapterDir,
	}
	if opts.Epochs > 0 {
		args = append(args, "--iters", fmt.Sprintf("%d", opts.Epochs*100))
	}

	logf, err := os.Create(logFile)
	if err != nil {
		return JobInfo{}, fmt.Errorf("mlx: create log: %w", err)
	}
	cmd := exec.Command(r.LoRACommand, args...)
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := cmd.Start(); err != nil {
		_ = logf.Close()
		return JobInfo{}, fmt.Errorf("mlx: start %s: %w", r.LoRACommand, err)
	}

	info := JobInfo{
		ID:         file.ID,
		Status:     StatusRunning,
		BaseModel:  opts.BaseModel,
		OutputName: adapterDir,
	}
	if err := writeStatus(statusFile, info, cmd.Process.Pid); err != nil {
		// Don't kill the running fine-tune just because we failed to write a sidecar
		// — keep the JobInfo accurate by surfacing the warning in Logs.
		info.Logs = "warn: status sidecar write failed: " + err.Error()
	}

	// Detach: reap exit code in goroutine and update status sidecar.
	go func() {
		err := cmd.Wait()
		_ = logf.Close()
		final := JobInfo{ID: file.ID, BaseModel: opts.BaseModel, OutputName: adapterDir}
		if err == nil {
			final.Status = StatusSucceeded
		} else {
			final.Status = StatusFailed
			final.Error = err.Error()
		}
		_ = writeStatus(statusFile, final, 0)
	}()

	return info, nil
}

// Poll reads status.json from the job dir. jobID is sanitised to prevent
// path traversal (".." or absolute paths are rejected).
func (r *MLXRunner) Poll(_ context.Context, jobID string) (JobInfo, error) {
	if err := validateJobID(jobID); err != nil {
		return JobInfo{}, err
	}
	statusFile := filepath.Join(r.Workspace, jobID, "status.json")
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return JobInfo{}, fmt.Errorf("mlx poll: read status: %w", err)
	}
	var stored struct {
		Info JobInfo `json:"info"`
		PID  int     `json:"pid"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return JobInfo{}, fmt.Errorf("mlx poll: decode status: %w", err)
	}
	return stored.Info, nil
}

// validateJobID accepts only the random hex slugs Upload generates: lowercase
// hex characters, length 4-64. Anything else (paths, traversal, empty) errors.
func validateJobID(id string) error {
	if id == "" {
		return fmt.Errorf("mlx: empty jobID")
	}
	if len(id) < 4 || len(id) > 64 {
		return fmt.Errorf("mlx: jobID length out of range")
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return fmt.Errorf("mlx: jobID contains illegal char")
		}
	}
	return nil
}

// writeStatus writes the sidecar atomically: write to <path>.tmp then rename.
// Rename within the same directory is atomic on POSIX, so concurrent Poll
// readers always see a fully-formed file.
func writeStatus(path string, info JobInfo, pid int) error {
	payload := struct {
		Info JobInfo `json:"info"`
		PID  int     `json:"pid"`
	}{info, pid}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copy: open src: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copy: create dst: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

func randomSlug() (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
