package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// OpenAIRunner implements Runner against api.openai.com.
type OpenAIRunner struct {
	APIKey  string
	BaseURL string // default https://api.openai.com/v1
	HTTP    *http.Client
}

// NewOpenAIRunner reads the OPENAI_API_KEY env var if apiKey is empty.
func NewOpenAIRunner(apiKey string) *OpenAIRunner {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAIRunner{
		APIKey:  apiKey,
		BaseURL: "https://api.openai.com/v1",
		HTTP:    &http.Client{},
	}
}

func (r *OpenAIRunner) Name() string { return "openai" }

// Upload posts the JSONL with purpose=fine-tune and returns the file_id.
func (r *OpenAIRunner) Upload(ctx context.Context, jsonlPath string) (FileRef, error) {
	if r.APIKey == "" {
		return FileRef{}, fmt.Errorf("openai: missing API key")
	}
	f, err := os.Open(jsonlPath)
	if err != nil {
		return FileRef{}, fmt.Errorf("openai upload: open %s: %w", jsonlPath, err)
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("purpose", "fine-tune"); err != nil {
		return FileRef{}, err
	}
	part, err := mw.CreateFormFile("file", filepath.Base(jsonlPath))
	if err != nil {
		return FileRef{}, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return FileRef{}, err
	}
	if err := mw.Close(); err != nil {
		return FileRef{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.BaseURL+"/files", &body)
	if err != nil {
		return FileRef{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := r.HTTP.Do(req)
	if err != nil {
		return FileRef{}, fmt.Errorf("openai upload: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return FileRef{}, fmt.Errorf("openai upload: status %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return FileRef{}, fmt.Errorf("openai upload: decode: %w", err)
	}
	return FileRef{ID: out.ID, Path: jsonlPath}, nil
}

// CreateJob starts a fine-tuning job. Default base model is gpt-4o-mini.
func (r *OpenAIRunner) CreateJob(ctx context.Context, file FileRef, opts CreateJobOpts) (JobInfo, error) {
	if opts.BaseModel == "" {
		opts.BaseModel = "gpt-4o-mini-2024-07-18"
	}
	payload := map[string]any{
		"training_file": file.ID,
		"model":         opts.BaseModel,
	}
	if opts.Suffix != "" {
		payload["suffix"] = opts.Suffix
	}
	if opts.Epochs > 0 {
		payload["hyperparameters"] = map[string]any{"n_epochs": opts.Epochs}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return JobInfo{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.BaseURL+"/fine_tuning/jobs", bytes.NewReader(body))
	if err != nil {
		return JobInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTP.Do(req)
	if err != nil {
		return JobInfo{}, fmt.Errorf("openai create job: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return JobInfo{}, fmt.Errorf("openai create job: status %d: %s", resp.StatusCode, string(respBody))
	}
	return decodeOpenAIJob(respBody)
}

// Poll fetches the latest state for a job.
func (r *OpenAIRunner) Poll(ctx context.Context, jobID string) (JobInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.BaseURL+"/fine_tuning/jobs/"+jobID, nil)
	if err != nil {
		return JobInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.APIKey)

	resp, err := r.HTTP.Do(req)
	if err != nil {
		return JobInfo{}, fmt.Errorf("openai poll: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return JobInfo{}, fmt.Errorf("openai poll: status %d: %s", resp.StatusCode, string(respBody))
	}
	return decodeOpenAIJob(respBody)
}

func decodeOpenAIJob(body []byte) (JobInfo, error) {
	var raw struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		Model         string `json:"model"`
		FineTunedModel string `json:"fine_tuned_model"`
		Error         struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return JobInfo{}, fmt.Errorf("openai decode job: %w", err)
	}
	return JobInfo{
		ID:         raw.ID,
		Status:     mapOpenAIStatus(raw.Status),
		BaseModel:  raw.Model,
		OutputName: raw.FineTunedModel,
		Error:      raw.Error.Message,
	}, nil
}

func mapOpenAIStatus(s string) Status {
	switch s {
	case "validating_files", "queued":
		return StatusQueued
	case "running":
		return StatusRunning
	case "succeeded":
		return StatusSucceeded
	case "failed":
		return StatusFailed
	case "cancelled":
		return StatusCancelled
	}
	return StatusUnknown
}
