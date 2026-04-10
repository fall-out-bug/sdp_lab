package extract

import "sdp_dev/internal/architect"

// DefaultExtractors returns the standard set of extractors for analysis.
func DefaultExtractors() []architect.Extractor {
	return []architect.Extractor{
		FileTreeExtractor{},
		DependencyManifestParser{},
		SpecInventoryScanner{},
		GeneratedCodeDetector{},
		&InfraExtractor{},
		GitHistoryExtractor{},
		GoAdapter{},
		PythonAdapter{},
		JavaAdapter{},
		TypeScriptAdapter{},
		&SQLExtractor{},
	}
}
