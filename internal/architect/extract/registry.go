package extract

import (
	"sdp_dev/internal/architect"
	"sdp_dev/internal/architect/extract/sql"
	"sdp_dev/internal/architect/extract/typescript"
)

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
		NewPythonAdapter{},
		NewJavaAdapter{},
		typescript.NewTSExtractor(),
		sql.NewSQLExtractor(),
	}
}
