package extract

import (
	"github.com/fall-out-bug/sdp_lab/internal/architect"
	"github.com/fall-out-bug/sdp_lab/internal/architect/extract/sql"
	"github.com/fall-out-bug/sdp_lab/internal/architect/extract/typescript"
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
