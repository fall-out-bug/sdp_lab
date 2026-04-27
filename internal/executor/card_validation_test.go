package executor

import (
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/control"
)

func TestValidateFeatureCard_NilCard(t *testing.T) {
	err := ValidateFeatureCard(nil)
	if err == nil {
		t.Fatalf("expected error for nil card, got nil")
	}
	if err.Error() != "card is nil" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateFeatureCard_AllRequiredFieldsPresent(t *testing.T) {
	card := &control.FeatureCard{
		ID:               "feature-123",
		Title:            "Add validation helper",
		NormalizedIntent: "Implement FeatureCard validation",
	}

	err := ValidateFeatureCard(card)
	if err != nil {
		t.Fatalf("expected no error for valid card, got: %v", err)
	}
}

func TestValidateFeatureCard_MissingID(t *testing.T) {
	card := &control.FeatureCard{
		Title:            "Add validation helper",
		NormalizedIntent: "Implement FeatureCard validation",
	}

	err := ValidateFeatureCard(card)
	if err == nil {
		t.Fatalf("expected error for missing ID, got nil")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError type, got %T", err)
	}

	if len(ve.MissingFields) != 1 || ve.MissingFields[0] != "ID" {
		t.Fatalf("expected missing ID field, got: %v", ve.MissingFields)
	}
}

func TestValidateFeatureCard_MissingTitle(t *testing.T) {
	card := &control.FeatureCard{
		ID:               "feature-123",
		NormalizedIntent: "Implement FeatureCard validation",
	}

	err := ValidateFeatureCard(card)
	if err == nil {
		t.Fatalf("expected error for missing Title, got nil")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError type, got %T", err)
	}

	if len(ve.MissingFields) != 1 || ve.MissingFields[0] != "Title" {
		t.Fatalf("expected missing Title field, got: %v", ve.MissingFields)
	}
}

func TestValidateFeatureCard_MissingNormalizedIntent(t *testing.T) {
	card := &control.FeatureCard{
		ID:    "feature-123",
		Title: "Add validation helper",
	}

	err := ValidateFeatureCard(card)
	if err == nil {
		t.Fatalf("expected error for missing NormalizedIntent, got nil")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError type, got %T", err)
	}

	if len(ve.MissingFields) != 1 || ve.MissingFields[0] != "NormalizedIntent" {
		t.Fatalf("expected missing NormalizedIntent field, got: %v", ve.MissingFields)
	}
}

func TestValidateFeatureCard_MultipleMissingFields(t *testing.T) {
	card := &control.FeatureCard{
		ID: "feature-123",
	}

	err := ValidateFeatureCard(card)
	if err == nil {
		t.Fatalf("expected error for missing multiple fields, got nil")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError type, got %T", err)
	}

	if len(ve.MissingFields) != 2 {
		t.Fatalf("expected 2 missing fields, got %d: %v", len(ve.MissingFields), ve.MissingFields)
	}
}

func TestValidateFeatureCard_WhitespaceOnlyFields(t *testing.T) {
	card := &control.FeatureCard{
		ID:               "   ",
		Title:            "   ",
		NormalizedIntent: "   ",
	}

	err := ValidateFeatureCard(card)
	if err == nil {
		t.Fatalf("expected error for whitespace-only fields, got nil")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError type, got %T", err)
	}

	if len(ve.MissingFields) != 3 {
		t.Fatalf("expected all 3 fields to be reported as missing, got %d: %v", len(ve.MissingFields), ve.MissingFields)
	}
}

func TestValidateFeatureCard_ErrorMessage(t *testing.T) {
	card := &control.FeatureCard{
		ID:               "feature-123",
		Title:            "",
		NormalizedIntent: "test intent",
	}

	err := ValidateFeatureCard(card)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	expectedMsg := "missing required field: Title"
	if err.Error() != expectedMsg {
		t.Fatalf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}
