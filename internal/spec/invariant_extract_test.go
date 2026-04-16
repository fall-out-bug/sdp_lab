package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractInvariants_InterfaceCompliance(t *testing.T) {
	inv, err := ExtractInvariants("testdata")
	require.NoError(t, err)
	var found bool
	for _, a := range inv.Architectural {
		if a.Category == "interface_compliance" {
			assert.Contains(t, a.Detail, "must implement")
			found = true
		}
	}
	assert.True(t, found, "should find at least one interface compliance check")
}

func TestExtractInvariants_MutexGuards(t *testing.T) {
	inv, err := ExtractInvariants("testdata")
	require.NoError(t, err)
	var found bool
	for _, c := range inv.Concurrency {
		if c.Category == "mutex_guard" {
			assert.Contains(t, c.Detail, "mutex")
			found = true
		}
	}
	assert.True(t, found, "should find at least one mutex guard")
}

func TestExtractInvariants_TypeAssertions(t *testing.T) {
	inv, err := ExtractInvariants("testdata")
	require.NoError(t, err)
	var found bool
	for _, ty := range inv.TypeSystem {
		if ty.Category == "type_assertion" {
			assert.Contains(t, ty.Detail, "assert to")
			found = true
		}
	}
	assert.True(t, found, "should find at least one type assertion")
}

func TestExtractInvariants_ContextDeadlines(t *testing.T) {
	inv, err := ExtractInvariants("testdata")
	require.NoError(t, err)
	var found bool
	for _, d := range inv.Database {
		if d.Constraint == "operation_timeout" {
			assert.Contains(t, d.Detail, "context.")
			found = true
		}
	}
	assert.True(t, found, "should find at least one context deadline")
}

func TestExtractInvariants_Total(t *testing.T) {
	inv, err := ExtractInvariants("testdata")
	require.NoError(t, err)
	expected := len(inv.Database) + len(inv.TypeSystem) + len(inv.Concurrency) + len(inv.Architectural)
	assert.Equal(t, expected, inv.Total)
}

func TestExtractInvariants_BadDir(t *testing.T) {
	inv, err := ExtractInvariants("testdata/nonexistent_xyz")
	require.NoError(t, err)
	assert.Equal(t, 0, inv.Total)
}
