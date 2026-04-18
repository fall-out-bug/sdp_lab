// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"testing"

	"sdp_dev/internal/architect"
)

// TestDetectPII tests the detectPII function with various table schemas.
func TestDetectPII(t *testing.T) {
	tests := []struct {
		name      string
		tables    []architect.Table
		wantCount int
		wantTypes map[string]bool // expected PII types to be detected
	}{
		{
			name: "user table with PII",
			tables: []architect.Table{
				{
					Name: "users",
					Columns: []architect.Column{
						{Name: "id", Type: "INT"},
						{Name: "email", Type: "VARCHAR(255)"},
						{Name: "phone", Type: "VARCHAR(20)"},
						{Name: "ssn", Type: "VARCHAR(11)"},
					},
				},
			},
			wantCount: 3,
			wantTypes: map[string]bool{
				"email_address":        true,
				"phone_number":         true,
				"social_security_number": true,
			},
		},
		{
			name: "table with partial PII matches",
			tables: []architect.Table{
				{
					Name: "customers",
					Columns: []architect.Column{
						{Name: "id", Type: "INT"},
						{Name: "primary_email", Type: "VARCHAR(255)"},
						{Name: "work_phone", Type: "VARCHAR(20)"},
						{Name: "home_address", Type: "TEXT"},
					},
				},
			},
			wantCount: 3,
			wantTypes: map[string]bool{
				"email_address":    true,
				"phone_number":     true,
				"physical_address": true,
			},
		},
		{
			name: "table with mixed PII columns",
			tables: []architect.Table{
				{
					Name: "profiles",
					Columns: []architect.Column{
						{Name: "id", Type: "INT"},
						{Name: "first_name", Type: "VARCHAR(100)"},
						{Name: "last_name", Type: "VARCHAR(100)"},
						{Name: "birth_date", Type: "DATE"},
						{Name: "zip_code", Type: "VARCHAR(10)"},
						{Name: "credit_card", Type: "VARCHAR(20)"},
					},
				},
			},
			wantCount: 5,
			wantTypes: map[string]bool{
				"personal_name":         true,
				"date_of_birth":         true,
				"location_identifier":   true,
				"financial_identifier":  true,
			},
		},
		{
			name: "table with no PII",
			tables: []architect.Table{
				{
					Name: "products",
					Columns: []architect.Column{
						{Name: "id", Type: "INT"},
						{Name: "name", Type: "VARCHAR(255)"},
						{Name: "price", Type: "DECIMAL(10,2)"},
						{Name: "created_at", Type: "TIMESTAMP"},
					},
				},
			},
			wantCount: 0,
			wantTypes: map[string]bool{},
		},
		{
			name: "multiple tables with PII",
			tables: []architect.Table{
				{
					Name: "users",
					Columns: []architect.Column{
						{Name: "id", Type: "INT"},
						{Name: "email", Type: "VARCHAR(255)"},
					},
				},
				{
					Name: "orders",
					Columns: []architect.Column{
						{Name: "id", Type: "INT"},
						{Name: "user_id", Type: "INT"},
						{Name: "shipping_address", Type: "TEXT"},
					},
				},
			},
			wantCount: 2,
			wantTypes: map[string]bool{
				"email_address":    true,
				"physical_address": true,
			},
		},
		{
			name: "table with government identifiers",
			tables: []architect.Table{
				{
					Name: "citizens",
					Columns: []architect.Column{
						{Name: "id", Type: "INT"},
						{Name: "passport", Type: "VARCHAR(20)"},
						{Name: "national_id", Type: "VARCHAR(20)"},
						{Name: "driver_license", Type: "VARCHAR(20)"},
					},
				},
			},
			wantCount: 3,
			wantTypes: map[string]bool{
				"government_identifier": true,
			},
		},
		{
			name: "table with exact and partial matches",
			tables: []architect.Table{
				{
					Name: "contacts",
					Columns: []architect.Column{
						{Name: "id", Type: "INT"},
						{Name: "email", Type: "VARCHAR(255)"},           // exact match
						{Name: "secondary_email", Type: "VARCHAR(255)"},  // partial match
						{Name: "email_address", Type: "VARCHAR(255)"},    // partial match
					},
				},
			},
			wantCount: 3,
			wantTypes: map[string]bool{
				"email_address": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			piiCols := detectPII(tt.tables)

			if len(piiCols) != tt.wantCount {
				t.Errorf("detectPII() returned %d columns, want %d", len(piiCols), tt.wantCount)
			}

			// Check that expected PII types are present
			foundTypes := make(map[string]bool)
			for _, p := range piiCols {
				foundTypes[p.PIIType] = true

				// Verify confidence levels
				if p.Confidence < 0.7 || p.Confidence > 1.0 {
					t.Errorf("detectPII() returned confidence %.2f for %s.%s, want between 0.7 and 1.0",
						p.Confidence, p.Table, p.Column)
				}

				// Exact matches should have higher confidence
				colLower := p.Column
				for _, pat := range piiExactPatterns {
					if colLower == pat && p.Confidence < 0.9 {
						t.Errorf("detectPII() exact match %s should have confidence >= 0.9, got %.2f",
							p.Column, p.Confidence)
					}
				}
			}

			for piiType := range tt.wantTypes {
				if !foundTypes[piiType] {
					t.Errorf("detectPII() did not find PII type %s", piiType)
				}
			}
		})
	}
}

// TestPIIColumnAttributes tests that detected PII columns have correct attributes.
func TestPIIColumnAttributes(t *testing.T) {
	tables := []architect.Table{
		{
			Name: "users",
			Columns: []architect.Column{
				{Name: "id", Type: "INT"},
				{Name: "email", Type: "VARCHAR(255)"},
				{Name: "primary_email", Type: "VARCHAR(255)"},
			},
		},
	}

	piiCols := detectPII(tables)

	if len(piiCols) != 2 {
		t.Fatalf("detectPII() returned %d columns, want 2", len(piiCols))
	}

	// Check exact match (email)
	var emailPII *architect.PIIColumn
	for i := range piiCols {
		if piiCols[i].Column == "email" {
			emailPII = &piiCols[i]
			break
		}
	}

	if emailPII == nil {
		t.Fatal("detectPII() did not find exact match for 'email'")
	}

	if emailPII.Confidence != 0.95 {
		t.Errorf("Exact match confidence = %.2f, want 0.95", emailPII.Confidence)
	}

	if emailPII.PIIType != "email_address" {
		t.Errorf("PII type = %s, want email_address", emailPII.PIIType)
	}

	if emailPII.Pattern != "email" {
		t.Errorf("Pattern = %s, want email", emailPII.Pattern)
	}

	// Check partial match (primary_email)
	var primaryEmailPII *architect.PIIColumn
	for i := range piiCols {
		if piiCols[i].Column == "primary_email" {
			primaryEmailPII = &piiCols[i]
			break
		}
	}

	if primaryEmailPII == nil {
		t.Fatal("detectPII() did not find partial match for 'primary_email'")
	}

	if primaryEmailPII.Confidence != 0.75 {
		t.Errorf("Partial match confidence = %.2f, want 0.75", primaryEmailPII.Confidence)
	}
}

// TestGroupPIIByType tests the GroupPIIByType function.
func TestGroupPIIByType(t *testing.T) {
	piiCols := []architect.PIIColumn{
		{Table: "users", Column: "email", PIIType: "email_address"},
		{Table: "users", Column: "phone", PIIType: "phone_number"},
		{Table: "customers", Column: "work_email", PIIType: "email_address"},
		{Table: "customers", Column: "mobile", PIIType: "phone_number"},
	}

	groups := GroupPIIByType(piiCols)

	if len(groups) != 2 {
		t.Errorf("GroupPIIByType() returned %d groups, want 2", len(groups))
	}

	if len(groups["email_address"]) != 2 {
		t.Errorf("GroupPIIByType() email_address group has %d items, want 2", len(groups["email_address"]))
	}

	if len(groups["phone_number"]) != 2 {
		t.Errorf("GroupPIIByType() phone_number group has %d items, want 2", len(groups["phone_number"]))
	}
}

// TestGroupPIIByTable tests the GroupPIIByTable function.
func TestGroupPIIByTable(t *testing.T) {
	piiCols := []architect.PIIColumn{
		{Table: "users", Column: "email", PIIType: "email_address"},
		{Table: "users", Column: "phone", PIIType: "phone_number"},
		{Table: "customers", Column: "work_email", PIIType: "email_address"},
	}

	groups := GroupPIIByTable(piiCols)

	if len(groups) != 2 {
		t.Errorf("GroupPIIByTable() returned %d groups, want 2", len(groups))
	}

	if len(groups["users"]) != 2 {
		t.Errorf("GroupPIIByTable() users group has %d items, want 2", len(groups["users"]))
	}

	if len(groups["customers"]) != 1 {
		t.Errorf("GroupPIIByTable() customers group has %d items, want 1", len(groups["customers"]))
	}
}

// TestFilterPIIByConfidence tests the FilterPIIByConfidence function.
func TestFilterPIIByConfidence(t *testing.T) {
	piiCols := []architect.PIIColumn{
		{Table: "users", Column: "email", Confidence: 0.95},
		{Table: "users", Column: "work_email", Confidence: 0.75},
		{Table: "users", Column: "primary_email", Confidence: 0.75},
	}

	// Test high confidence filter
	highConf := FilterPIIByConfidence(piiCols, 0.9)
	if len(highConf) != 1 {
		t.Errorf("FilterPIIByConfidence(0.9) returned %d items, want 1", len(highConf))
	}

	// Test medium confidence filter
	medConf := FilterPIIByConfidence(piiCols, 0.7)
	if len(medConf) != 3 {
		t.Errorf("FilterPIIByConfidence(0.7) returned %d items, want 3", len(medConf))
	}

	// Test very high confidence filter
	veryHighConf := FilterPIIByConfidence(piiCols, 0.99)
	if len(veryHighConf) != 0 {
		t.Errorf("FilterPIIByConfidence(0.99) returned %d items, want 0", len(veryHighConf))
	}
}

// TestFilterPIIByType tests the FilterPIIByType function.
func TestFilterPIIByType(t *testing.T) {
	piiCols := []architect.PIIColumn{
		{Table: "users", Column: "email", PIIType: "email_address"},
		{Table: "users", Column: "phone", PIIType: "phone_number"},
		{Table: "customers", Column: "work_email", PIIType: "email_address"},
	}

	emailPII := FilterPIIByType(piiCols, "email_address")
	if len(emailPII) != 2 {
		t.Errorf("FilterPIIByType(email_address) returned %d items, want 2", len(emailPII))
	}

	phonePII := FilterPIIByType(piiCols, "phone_number")
	if len(phonePII) != 1 {
		t.Errorf("FilterPIIByType(phone_number) returned %d items, want 1", len(phonePII))
	}

	unknownPII := FilterPIIByType(piiCols, "unknown_type")
	if len(unknownPII) != 0 {
		t.Errorf("FilterPIIByType(unknown_type) returned %d items, want 0", len(unknownPII))
	}
}

// TestFilterPIIByTable tests the FilterPIIByTable function.
func TestFilterPIIByTable(t *testing.T) {
	piiCols := []architect.PIIColumn{
		{Table: "users", Column: "email", PIIType: "email_address"},
		{Table: "users", Column: "phone", PIIType: "phone_number"},
		{Table: "customers", Column: "work_email", PIIType: "email_address"},
	}

	userPII := FilterPIIByTable(piiCols, "users")
	if len(userPII) != 2 {
		t.Errorf("FilterPIIByTable(users) returned %d items, want 2", len(userPII))
	}

	customerPII := FilterPIIByTable(piiCols, "customers")
	if len(customerPII) != 1 {
		t.Errorf("FilterPIIByTable(customers) returned %d items, want 1", len(customerPII))
	}
}

// TestGetPIIStats tests the GetPIIStats function.
func TestGetPIIStats(t *testing.T) {
	piiCols := []architect.PIIColumn{
		{Table: "users", Column: "email", Confidence: 0.95},
		{Table: "users", Column: "phone", Confidence: 0.75},
		{Table: "users", Column: "work_email", Confidence: 0.75},
		{Table: "customers", Column: "address", Confidence: 0.75},
	}

	stats := GetPIIStats(piiCols)

	if stats["total"] != 4 {
		t.Errorf("GetPIIStats() total = %d, want 4", stats["total"])
	}

	if stats["high_conf"] != 1 {
		t.Errorf("GetPIIStats() high_conf = %d, want 1", stats["high_conf"])
	}

	if stats["medium_conf"] != 3 {
		t.Errorf("GetPIIStats() medium_conf = %d, want 3", stats["medium_conf"])
	}

	if stats["low_conf"] != 0 {
		t.Errorf("GetPIIStats() low_conf = %d, want 0", stats["low_conf"])
	}
}

// TestCheckPIIEncryption tests the CheckPIIEncryption function.
func TestCheckPIIEncryption(t *testing.T) {
	tests := []struct {
		name     string
		column   architect.Column
		wantBool bool
	}{
		{
			name: "encrypted column name",
			column: architect.Column{
				Name: "encrypted_email",
				Type: "VARCHAR(255)",
			},
			wantBool: true,
		},
		{
			name: "hashed column name",
			column: architect.Column{
				Name: "password_hash",
				Type: "VARCHAR(255)",
			},
			wantBool: true,
		},
		{
			name: "binary column type",
			column: architect.Column{
				Name: "secret",
				Type: "VARBINARY(255)",
			},
			wantBool: true,
		},
		{
			name: "blob column type",
			column: architect.Column{
				Name: "data",
				Type: "BLOB",
			},
			wantBool: true,
		},
		{
			name: "normal column",
			column: architect.Column{
				Name: "email",
				Type: "VARCHAR(255)",
			},
			wantBool: false,
		},
		{
			name: "bcrypt column name",
			column: architect.Column{
				Name: "password_bcrypt",
				Type: "VARCHAR(255)",
			},
			wantBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPIIEncryption(tt.column)
			if result != tt.wantBool {
				t.Errorf("CheckPIIEncryption() = %v, want %v", result, tt.wantBool)
			}
		})
	}
}

// TestEnrichPIIWithEncryptionDetection tests the EnrichPIIWithEncryptionDetection function.
func TestEnrichPIIWithEncryptionDetection(t *testing.T) {
	tables := []architect.Table{
		{
			Name: "users",
			Columns: []architect.Column{
				{Name: "id", Type: "INT"},
				{Name: "email", Type: "VARCHAR(255)"},
				{Name: "encrypted_phone", Type: "VARBINARY(255)"},
			},
		},
	}

	piiCols := []architect.PIIColumn{
		{Table: "users", Column: "email", Confidence: 0.95},
		{Table: "users", Column: "encrypted_phone", Confidence: 0.75},
	}

	enriched := EnrichPIIWithEncryptionDetection(piiCols, tables)

	if len(enriched) != 2 {
		t.Fatalf("EnrichPIIWithEncryptionDetection() returned %d items, want 2", len(enriched))
	}

	// Find email column (should not be encrypted)
	var emailPII *architect.PIIColumn
	for i := range enriched {
		if enriched[i].Column == "email" {
			emailPII = &enriched[i]
			break
		}
	}

	if emailPII == nil {
		t.Fatal("EnrichPIIWithEncryptionDetection() did not find email column")
	}

	if emailPII.EncryptionDetected {
		t.Error("Email column should not have encryption detected")
	}

	// Find encrypted_phone column (should be encrypted)
	var phonePII *architect.PIIColumn
	for i := range enriched {
		if enriched[i].Column == "encrypted_phone" {
			phonePII = &enriched[i]
			break
		}
	}

	if phonePII == nil {
		t.Fatal("EnrichPIIWithEncryptionDetection() did not find encrypted_phone column")
	}

	if !phonePII.EncryptionDetected {
		t.Error("Encrypted phone column should have encryption detected")
	}
}
