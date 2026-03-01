// Package beads provides integration with the Beads issue tracking system.
package beads

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Formula represents a Beads workflow formula.
type Formula struct {
	// Name is the formula identifier.
	Name string `json:"name" yaml:"name"`
	
	// Description explains what the formula does.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	
	// Version is the formula version.
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	
	// Variables are the formula's configurable parameters.
	Variables map[string]Variable `json:"variables,omitempty" yaml:"variables,omitempty"`
	
	// Steps are the workflow steps.
	Steps []FormulaStep `json:"steps" yaml:"steps"`
	
	// Extends is the parent formula to inherit from.
	Extends string `json:"extends,omitempty" yaml:"extends,omitempty"`
	
	// Compose combines multiple formulas.
	Compose []string `json:"compose,omitempty" yaml:"compose,omitempty"`
	
	// Aspects are cross-cutting concerns to apply.
	Aspects []string `json:"aspects,omitempty" yaml:"aspects,omitempty"`
	
	// Dependencies are formulas that must complete first.
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	
	// SourcePath is where the formula was loaded from.
	SourcePath string `json:"-" yaml:"-"`
}

// Variable represents a formula variable.
type Variable struct {
	// Description explains the variable.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	
	// Type is the variable type (string, int, bool, list).
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	
	// Default is the default value.
	Default interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	
	// Required indicates if the variable must be provided.
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`
	
	// Enum lists allowed values.
	Enum []string `json:"enum,omitempty" yaml:"enum,omitempty"`
}

// FormulaStep represents a single step in a formula.
type FormulaStep struct {
	// Name is the step identifier.
	Name string `json:"name" yaml:"name"`
	
	// Title is the human-readable step title.
	Title string `json:"title,omitempty" yaml:"title,omitempty"`
	
	// Description explains what the step does.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	
	// Type is the step type (task, feature, bug, etc.).
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	
	// Priority is the step priority (1-3).
	Priority int `json:"priority,omitempty" yaml:"priority,omitempty"`
	
	// Size is the estimated size (S, M, L, XL).
	Size string `json:"size,omitempty" yaml:"size,omitempty"`
	
	// Dependencies are step IDs that must complete first.
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	
	// ScopeFiles are files the step is allowed to modify.
	ScopeFiles []string `json:"scope_files,omitempty" yaml:"scope_files,omitempty"`
	
	// AcceptanceCriteria are the step's AC.
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty" yaml:"acceptance_criteria,omitempty"`
	
	// Variables are step-specific variable overrides.
	Variables map[string]interface{} `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// FormulaParser parses Beads formula files.
type FormulaParser struct {
	searchPaths []string
}

// NewFormulaParser creates a new formula parser.
func NewFormulaParser() *FormulaParser {
	return &FormulaParser{
		searchPaths: []string{
			".beads/formulas",
			filepath.Join(os.Getenv("HOME"), ".beads/formulas"),
		},
	}
}

// AddSearchPath adds a search path for formulas.
func (p *FormulaParser) AddSearchPath(path string) {
	p.searchPaths = append(p.searchPaths, path)
}

// ParseFile parses a formula file (YAML or JSON).
func (p *FormulaParser) ParseFile(path string) (*Formula, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read formula: %w", err)
	}
	
	return p.Parse(data, path)
}

// Parse parses formula data.
func (p *FormulaParser) Parse(data []byte, sourcePath string) (*Formula, error) {
	var formula Formula
	
	// Try YAML first, then JSON
	if err := yaml.Unmarshal(data, &formula); err != nil {
		if jsonErr := json.Unmarshal(data, &formula); jsonErr != nil {
			return nil, fmt.Errorf("parse formula (tried yaml and json): yaml: %v, json: %v", err, jsonErr)
		}
	}
	
	if formula.Name == "" {
		return nil, fmt.Errorf("formula missing name")
	}
	
	if len(formula.Steps) == 0 {
		return nil, fmt.Errorf("formula %q has no steps", formula.Name)
	}
	
	formula.SourcePath = sourcePath
	
	return &formula, nil
}

// FindFormula searches for a formula by name.
func (p *FormulaParser) FindFormula(name string) (*Formula, error) {
	for _, searchPath := range p.searchPaths {
		// Try YAML first
		yamlPath := filepath.Join(searchPath, name+".yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			return p.ParseFile(yamlPath)
		}
		
		// Try yml extension
		ymlPath := filepath.Join(searchPath, name+".yml")
		if _, err := os.Stat(ymlPath); err == nil {
			return p.ParseFile(ymlPath)
		}
		
		// Try JSON
		jsonPath := filepath.Join(searchPath, name+".json")
		if _, err := os.Stat(jsonPath); err == nil {
			return p.ParseFile(jsonPath)
		}
	}
	
	return nil, fmt.Errorf("formula %q not found in search paths", name)
}

// ListFormulas returns all available formulas.
func (p *FormulaParser) ListFormulas() ([]*Formula, error) {
	var formulas []*Formula
	seen := make(map[string]bool)
	
	for _, searchPath := range p.searchPaths {
		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue
		}
		
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			
			name := entry.Name()
			ext := filepath.Ext(name)
			if ext != ".yaml" && ext != ".yml" && ext != ".json" {
				continue
			}
			
			baseName := strings.TrimSuffix(name, ext)
			if seen[baseName] {
				continue
			}
			
			path := filepath.Join(searchPath, name)
			formula, err := p.ParseFile(path)
			if err != nil {
				continue
			}
			
			seen[baseName] = true
			formulas = append(formulas, formula)
		}
	}
	
	return formulas, nil
}

// Hash returns a hash of the formula for versioning.
func (f *Formula) Hash() string {
	// Simple hash based on name, version, and step count
	data := fmt.Sprintf("%s:%s:%d", f.Name, f.Version, len(f.Steps))
	return fmt.Sprintf("%x", len(data))[:8]
}
