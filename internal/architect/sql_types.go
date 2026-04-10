package architect

// TableDef represents a database table definition.
type TableDef struct {
	Name    string      `json:"name"`
	Columns []ColumnDef `json:"columns,omitempty"`
	Schema  string      `json:"schema,omitempty"`
}

// ColumnDef represents a column within a table.
type ColumnDef struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable,omitempty"`
	PrimaryKey bool `json:"primary_key,omitempty"`
}

// ForeignKey represents a foreign key relationship between tables.
type ForeignKey struct {
	FromTable  string `json:"from_table"`
	FromColumn string `json:"from_column"`
	ToTable    string `json:"to_table"`
	ToColumn   string `json:"to_column"`
}

// ViewDef represents a database view.
type ViewDef struct {
	Name       string `json:"name"`
	Definition string `json:"definition,omitempty"`
}

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

// IndexDef represents a database index.
type IndexDef struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique,omitempty"`
}

// DataDomain groups tables into a logical domain based on FK clustering.
type DataDomain struct {
	Name   string   `json:"name"`
	Tables []string `json:"tables"`
}

// PIIColumn flags a column that likely contains personally identifiable information.
type PIIColumn struct {
	Table               string  `json:"table"`
	Column              string  `json:"column"`
	PIIType             string  `json:"pii_type"`              // "email_address", "phone_number", "date_of_birth", "ssn", etc.
	Confidence          float64 `json:"confidence"`
	EncryptionDetected  bool    `json:"encryption_detected"`
	Recommendation      string  `json:"recommendation,omitempty"`
}

// SQLAnalysis holds the full data architecture analysis.
type SQLAnalysis struct {
	DatabasesDetected int          `json:"databases_detected"`
	Tables            []TableDef   `json:"tables,omitempty"`
	ForeignKeys       []ForeignKey `json:"foreign_keys,omitempty"`
	Views             []ViewDef    `json:"views,omitempty"`
	Migrations        []Migration  `json:"migrations,omitempty"`
	MigrationsCount   int          `json:"migrations_count"`
	LatestMigration   string       `json:"latest_migration,omitempty"`
	StoredProcs       []StoredProc `json:"stored_procs,omitempty"`
	Indexes           []IndexDef   `json:"indexes,omitempty"`
	DataDomains       []DataDomain `json:"data_domains,omitempty"`
	PIIColumns        []PIIColumn  `json:"pii_columns,omitempty"`
}
