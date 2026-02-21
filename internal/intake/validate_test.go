package intake

import "testing"

func TestValidateProjectID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"", true},
		{"default", true},
		{"my-project", true},
		{"my_project", true},
		{"Project123", true},
		{"a", true},
		{"project.with.dots", false},
		{"project>wildcard", false},
		{"project*star", false},
		{"has space", false},
		{"has\ttab", false},
		{"unicode-日本語", false},
		{string(make([]byte, 129)), false},
	}
	for _, tt := range tests {
		err := ValidateProjectID(tt.id)
		got := err == nil
		if got != tt.want {
			t.Errorf("ValidateProjectID(%q) err=%v want valid=%v", tt.id, err, tt.want)
		}
	}
}
