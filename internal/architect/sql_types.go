package architect

// Column represents a column within a table.
type Column struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable,omitempty"`
	PrimaryKey bool   `json:"primary_key,omitempty"`
	NotNull    bool   `json:"not_null,omitempty"`
}

// Table represents a database table definition.
type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns,omitempty"`
	Schema  string   `json:"schema,omitempty"`
	File    string   `json:"file,omitempty"`
}

// TableDef is an alias for Table (kept for backward compatibility).
type TableDef = Table

// ColumnDef is an alias for Column (kept for backward compatibility).
type ColumnDef = Column

// ForeignKey represents a foreign key relationship between tables.
type ForeignKey struct {
	FromTable  string `json:"from_table"`
	FromColumn string `json:"from_column"`
	ToTable    string `json:"to_table"`
	ToColumn   string `json:"to_column"`
	File       string `json:"file,omitempty"`
}

// View represents a database view.
type View struct {
	Name         string `json:"name"`
	Definition   string `json:"definition,omitempty"`
	Materialized bool   `json:"materialized,omitempty"`
	File         string `json:"file,omitempty"`
}

// ViewDef is an alias for View (kept for backward compatibility).
type ViewDef = View

// Migration represents a database migration file.
type Migration struct {
	Path      string `json:"path"`
	Version   string `json:"version,omitempty"`
	Direction string `json:"direction,omitempty"` // "up", "down"
}

// StoredProc represents a stored procedure or function.
type StoredProc struct {
	Name     string `json:"name"`
	Path     string `json:"path,omitempty"`
	Language string `json:"language,omitempty"` // "plpgsql", "tsql", etc.
}

// Index represents a database index.
type Index struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique,omitempty"`
	File    string   `json:"file,omitempty"`
}

// IndexDef is an alias for Index (kept for backward compatibility).
type IndexDef = Index

// DataDomain groups tables into a logical domain based on FK clustering.
type DataDomain struct {
	Name   string   `json:"name"`
	Tables []string `json:"tables"`
}

// PIIColumn flags a column that likely contains personally identifiable information.
type PIIColumn struct {
	Table               string  `json:"table"`
	Column              string  `json:"column"`
	PIIType             string  `json:"pii_type,omitempty"`    // "email_address", "phone_number", etc.
	Pattern             string  `json:"pattern,omitempty"`     // matched PII pattern
	Confidence          float64 `json:"confidence"`
	EncryptionDetected  bool    `json:"encryption_detected,omitempty"`
	Recommendation      string  `json:"recommendation,omitempty"`
}

// SQLAnalysis holds the full data architecture analysis.
type SQLAnalysis struct {
	DatabasesDetected int            `json:"databases_detected"`
	Tables            []Table        `json:"tables,omitempty"`
	ForeignKeys       []ForeignKey   `json:"foreign_keys,omitempty"`
	Views             []View         `json:"views,omitempty"`
	Migrations        *MigrationInfo `json:"migrations,omitempty"`
	MigrationsCount   int            `json:"migrations_count"`
	LatestMigration   string         `json:"latest_migration,omitempty"`
	StoredProcs       []StoredProc   `json:"stored_procs,omitempty"`
	Indexes           []Index        `json:"indexes,omitempty"`
	DataDomains       []DataDomain   `json:"data_domains,omitempty"`
	PIIColumns        []PIIColumn    `json:"pii_columns,omitempty"`
	ORMModels         []ORMModel     `json:"orm_models,omitempty"`
}

// MigrationInfo describes detected database migrations.
type MigrationInfo struct {
	Dir    string `json:"dir"`
	Count  int    `json:"count"`
	Latest string `json:"latest,omitempty"`
}

// ORMModel records an ORM framework usage found in a source file.
type ORMModel struct {
	Framework string `json:"framework"` // "gorm", "django", "sqlalchemy", "prisma", "jpa"
	File      string `json:"file"`
	Model     string `json:"model,omitempty"`
}
