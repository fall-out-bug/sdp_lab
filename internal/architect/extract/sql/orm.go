// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"sdp_dev/internal/architect"
)

// ---------------------------------------------------------------------------
// ORM detection regexes (file-level)
// ---------------------------------------------------------------------------

var (
	reGORM           = regexp.MustCompile(`gorm\.Model`)
	reGORMTag        = regexp.MustCompile(`gorm:"`)
	reGORMStruct     = regexp.MustCompile(`(?m)type\s+(\w+)\s+struct\b`)
	reDjangoORM      = regexp.MustCompile(`models\.Model`)
	reDjangoORMModel = regexp.MustCompile(`(?m)class\s+(\w+)\s*\(\s*models\.Model\s*\)`)
	reSQLAlchemy     = regexp.MustCompile(`(?:Column\s*\(|declarative_base)`)
	reSAModelClass   = regexp.MustCompile(`(?m)class\s+(\w+)\s*\(\s*Base\s*\)`)
	rePrismaModel    = regexp.MustCompile(`(?m)^\s*model\s+(\w+)\s*\{`)
	reJPA            = regexp.MustCompile(`@(?:Entity|Table)`)
	reJPAClass       = regexp.MustCompile(`(?m)public\s+class\s+(\w+)`)
)

// detectORM detects ORM frameworks used in a source file.
func detectORM(content, file string) []architect.ORMModel {
	var models []architect.ORMModel

	ext := strings.ToLower(filepath.Ext(file))

	switch ext {
	case ".go":
		if reGORM.MatchString(content) || reGORMTag.MatchString(content) {
			// Try to extract struct names that use GORM.
			structNames := reGORMStruct.FindAllStringSubmatch(content, -1)
			if len(structNames) > 0 {
				for _, m := range structNames {
					models = append(models, architect.ORMModel{
						Framework: "gorm",
						File:      file,
						Model:     m[1],
					})
				}
			} else {
				models = append(models, architect.ORMModel{
					Framework: "gorm",
					File:      file,
				})
			}
		}
	case ".py":
		// Django models: class Foo(models.Model)
		if reDjangoORM.MatchString(content) {
			djangoMatches := reDjangoORMModel.FindAllStringSubmatch(content, -1)
			if len(djangoMatches) > 0 {
				for _, m := range djangoMatches {
					models = append(models, architect.ORMModel{
						Framework: "django",
						File:      file,
						Model:     m[1],
					})
				}
			} else {
				models = append(models, architect.ORMModel{
					Framework: "django",
					File:      file,
				})
			}
		}
		// SQLAlchemy models: class Foo(Base) or Column(...) usage
		if reSQLAlchemy.MatchString(content) {
			saMatches := reSAModelClass.FindAllStringSubmatch(content, -1)
			if len(saMatches) > 0 {
				for _, m := range saMatches {
					models = append(models, architect.ORMModel{
						Framework: "sqlalchemy",
						File:      file,
						Model:     m[1],
					})
				}
			} else {
				models = append(models, architect.ORMModel{
					Framework: "sqlalchemy",
					File:      file,
				})
			}
		}
	case ".prisma":
		// Prisma model names extracted directly from regex.
		for _, m := range rePrismaModel.FindAllStringSubmatch(content, -1) {
			models = append(models, architect.ORMModel{
				Framework: "prisma",
				File:      file,
				Model:     m[1],
			})
		}
	case ".java":
		if reJPA.MatchString(content) {
			// Try to extract the class name annotated with @Entity/@Table.
			classMatches := reJPAClass.FindAllStringSubmatch(content, -1)
			if len(classMatches) > 0 {
				for _, m := range classMatches {
					models = append(models, architect.ORMModel{
						Framework: "jpa",
						File:      file,
						Model:     m[1],
					})
				}
			} else {
				models = append(models, architect.ORMModel{
					Framework: "jpa",
					File:      file,
				})
			}
		}
	}

	return models
}

// DetectORMFrameworks detects all ORM frameworks used in a codebase.
func DetectORMFrameworks(models []architect.ORMModel) map[string]bool {
	frameworks := make(map[string]bool)
	for _, m := range models {
		frameworks[m.Framework] = true
	}
	return frameworks
}

// GroupModelsByFramework groups ORM models by their framework.
func GroupModelsByFramework(models []architect.ORMModel) map[string][]architect.ORMModel {
	groups := make(map[string][]architect.ORMModel)
	for _, m := range models {
		groups[m.Framework] = append(groups[m.Framework], m)
	}
	return groups
}

// SortModels sorts ORM models by framework then file for deterministic output.
func SortModels(models []architect.ORMModel) {
	sort.Slice(models, func(i, j int) bool {
		if models[i].Framework == models[j].Framework {
			if models[i].File == models[j].File {
				return models[i].Model < models[j].Model
			}
			return models[i].File < models[j].File
		}
		return models[i].Framework < models[j].Framework
	})
}

// GetFrameworkCount returns the count of models per framework.
func GetFrameworkCount(models []architect.ORMModel) map[string]int {
	counts := make(map[string]int)
	for _, m := range models {
		counts[m.Framework]++
	}
	return counts
}

// CorrelateORMWithTables attempts to correlate ORM models with SQL tables.
// Uses naming conventions: User model -> users table, UserProfile -> user_profiles, etc.
func CorrelateORMWithTables(models []architect.ORMModel, tables []architect.Table) map[string]string {
	correlations := make(map[string]string)

	tableNames := make(map[string]bool)
	for _, t := range tables {
		tableNames[t.Name] = true
	}

	for _, m := range models {
		if m.Model == "" {
			continue
		}

		// Try different naming conventions
		candidates := []string{
			m.Model,                     // Exact match: User -> User
			strings.ToLower(m.Model),    // Lowercase: User -> user
			toSnakeCase(m.Model),        // Snake case: UserProfile -> user_profile
			toSnakeCase(m.Model) + "s",  // Plural: User -> users
			strings.ToLower(m.Model) + "s", // Lower plural: User -> users
		}

		for _, candidate := range candidates {
			if tableNames[candidate] {
				correlations[m.Model] = candidate
				break
			}
		}
	}

	return correlations
}

// toSnakeCase converts CamelCase to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// DetectSchemaFiles detects schema definition files (schema.sql, schema.prisma, *.dbml).
func DetectSchemaFiles(root string) []string {
	var schemaFiles []string

	// Well-known schema file patterns
	schemaPatterns := []string{
		"schema.sql",
		"schema.prisma",
		"schema.dbml",
		"database.sql",
		"init.sql",
		"structure.sql",
	}

	for _, pattern := range schemaPatterns {
		path := filepath.Join(root, pattern)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			schemaFiles = append(schemaFiles, pattern)
		}
	}

	// Look for .dbml files in common directories
	dbmlDirs := []string{"docs", "database", "db", "schemas"}
	for _, dir := range dbmlDirs {
		dirPath := filepath.Join(root, dir)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			entries, err := os.ReadDir(dirPath)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".dbml") {
					schemaFiles = append(schemaFiles, filepath.Join(dir, e.Name()))
				}
			}
		}
	}

	return schemaFiles
}
