package control

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type OrchestrateOnceResult struct {
	Action  string `json:"action"`
	Message string `json:"message"`

	IngestedCard *FeatureCard `json:"ingested_card,omitempty"`

	DispatchedCard *FeatureCard `json:"dispatched_card,omitempty"`
	ExecutorRole   string       `json:"executor_role,omitempty"`
	PacketPath     string       `json:"packet_path,omitempty"`

	NoActionReason string `json:"no_action_reason,omitempty"`
}

func (s *Store) OrchestrateOnce() (*OrchestrateOnceResult, error) {
	if result, err := s.ingestExecutorResultIfExists(); err != nil {
		return nil, fmt.Errorf("ingest executor result: %w", err)
	} else if result != nil {
		return result, nil
	}

	dispatchResult, err := s.DispatchNext()
	if err != nil {
		return nil, fmt.Errorf("dispatch next: %w", err)
	}

	if dispatchResult.Success {
		card, err := s.LoadCard(dispatchResult.ProjectID, dispatchResult.CardID)
		if err != nil {
			return nil, fmt.Errorf("load dispatched card: %w", err)
		}
		return &OrchestrateOnceResult{
			Action:         "dispatched",
			Message:        dispatchResult.Message,
			DispatchedCard: card,
			ExecutorRole:   dispatchResult.ExecutorRole,
			PacketPath:     dispatchResult.PacketPath,
		}, nil
	}

	return &OrchestrateOnceResult{
		Action:         "no_action",
		Message:        "No orchestration action taken",
		NoActionReason: dispatchResult.NoDispatchableReason,
	}, nil
}

func (s *Store) executorResultsDir() string {
	return filepath.Join(s.ControlRoot, "executor-results")
}

func (s *Store) ingestedResultsFile() string {
	return filepath.Join(s.ControlRoot, "executor-results", ".ingested")
}

func (s *Store) ingestExecutorResultIfExists() (*OrchestrateOnceResult, error) {
	resultsDir := s.executorResultsDir()
	resultsFile := s.ingestedResultsFile()

	ingestedSet := make(map[string]struct{})
	if data, err := os.ReadFile(resultsFile); err == nil {
		lines := splitLines(string(data))
		for _, line := range lines {
			if line != "" {
				ingestedSet[filepath.Base(line)] = struct{}{}
			}
		}
	}

	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read executor results dir: %w", err)
	}

	var pending []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if _, ingested := ingestedSet[entry.Name()]; ingested {
			continue
		}
		if entry.Name()[0] == '.' {
			continue
		}
		pending = append(pending, entry.Name())
	}

	if len(pending) == 0 {
		return nil, nil
	}

	sort.Slice(pending, func(i, j int) bool {
		pathI := filepath.Join(resultsDir, pending[i])
		pathJ := filepath.Join(resultsDir, pending[j])
		infoI, errI := os.Stat(pathI)
		infoJ, errJ := os.Stat(pathJ)
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	resultPath := filepath.Join(resultsDir, pending[0])
	ingestedCard, err := s.ingestExecutorResultFile(resultPath)
	if err != nil {
		return nil, fmt.Errorf("ingest executor result from %s: %w", resultPath, err)
	}

	if err := s.markResultIngested(resultPath); err != nil {
		return nil, fmt.Errorf("mark result as ingested: %w", err)
	}

	return &OrchestrateOnceResult{
		Action:       "ingested",
		Message:      fmt.Sprintf("Ingested executor result for card %s", ingestedCard.ID),
		IngestedCard: ingestedCard,
	}, nil
}

func (s *Store) ingestExecutorResultFile(path string) (*FeatureCard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read result packet: %w", err)
	}

	packet, err := jsonUnmarshal(data)
	if err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("unmarshal result packet: %w", err)
	}

	card, err := s.IngestExecutorResult(packet)
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	_ = os.Remove(path)
	return card, nil
}

func (s *Store) markResultIngested(resultPath string) error {
	resultsFile := s.ingestedResultsFile()
	if err := os.MkdirAll(filepath.Dir(resultsFile), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(resultsFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var lines []string
	if err == nil {
		lines = splitLines(string(data))
	}

	lines = append(lines, resultPath)
	return writeLines(resultsFile, lines)
}

func splitLines(data string) []string {
	if data == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := data[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		line := data[start:]
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func writeLines(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return nil
}
