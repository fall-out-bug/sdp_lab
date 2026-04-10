package architect

// ExtractionResult is the unified output of language-specific extractors
// (Python, Java, TypeScript) that perform regex-based code analysis.
type ExtractionResult struct {
	Language         string       `json:"language"`
	ExtractionMethod string       `json:"extraction_method"`
	AccuracyEstimate float64      `json:"accuracy_estimate"`
	FileCount        int          `json:"file_count,omitempty"`
	Dependencies     []Dependency `json:"dependencies,omitempty"`
	Frameworks       []Framework  `json:"frameworks,omitempty"`
}

// Dependency represents a single resolved dependency from source analysis.
type Dependency struct {
	Name   string `json:"name"`
	Source string `json:"source,omitempty"` // e.g. "import", "requirements.txt", "pyproject.toml"
	Kind   string `json:"kind,omitempty"`   // "stdlib", "third-party", "relative"
}

// Framework describes a detected framework or annotation pattern.
type Framework struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
}
