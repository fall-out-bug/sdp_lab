package finetune

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteJSONL writes samples to path as JSONL (one Sample per line, no Meta).
func WriteJSONL(path string, samples []Sample) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("finetune: mkdir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("finetune: create %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, s := range samples {
		// Meta has json:"-", so it is not serialised.
		if err := enc.Encode(s); err != nil {
			return fmt.Errorf("finetune: encode sample: %w", err)
		}
	}
	return w.Flush()
}
