package dispatch

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSelectTiers_GroupsByTierClass(t *testing.T) {
	pFast := &CapabilityProfile{Harness: "h1", TierClass: TierFast}
	pStrong := &CapabilityProfile{Harness: "h2", TierClass: TierStrong}
	pBalanced := &CapabilityProfile{Harness: "h3", TierClass: TierBalanced}

	got := SelectTiers(
		[]*CapabilityProfile{pFast, pStrong, pBalanced},
		[]TierClass{TierFast, TierBalanced, TierStrong},
	)

	if len(got) != 4 { // 3 requested + untiered
		t.Fatalf("expected 4 buckets, got %d", len(got))
	}
	if !reflect.DeepEqual(got[0], []*CapabilityProfile{pFast}) {
		t.Errorf("fast bucket = %v, want [pFast]", got[0])
	}
	if !reflect.DeepEqual(got[1], []*CapabilityProfile{pBalanced}) {
		t.Errorf("balanced bucket = %v, want [pBalanced]", got[1])
	}
	if !reflect.DeepEqual(got[2], []*CapabilityProfile{pStrong}) {
		t.Errorf("strong bucket = %v, want [pStrong]", got[2])
	}
	if len(got[3]) != 0 {
		t.Errorf("untiered bucket should be empty, got %v", got[3])
	}
}

func TestSelectTiers_PreservesOrderWithinTier(t *testing.T) {
	a := &CapabilityProfile{Harness: "a", TierClass: TierFast}
	b := &CapabilityProfile{Harness: "b", TierClass: TierFast}
	c := &CapabilityProfile{Harness: "c", TierClass: TierFast}
	got := SelectTiers([]*CapabilityProfile{a, b, c}, []TierClass{TierFast})
	want := []*CapabilityProfile{a, b, c}
	if !reflect.DeepEqual(got[0], want) {
		t.Errorf("order not preserved: %v want %v", got[0], want)
	}
}

func TestSelectTiers_UntieredBucket(t *testing.T) {
	noTier := &CapabilityProfile{Harness: "x"}                      // empty TierClass
	bogus := &CapabilityProfile{Harness: "y", TierClass: "garbage"} // unrecognised
	got := SelectTiers([]*CapabilityProfile{noTier, bogus}, []TierClass{TierFast, TierStrong})
	if len(got[0]) != 0 || len(got[1]) != 0 {
		t.Errorf("requested buckets should be empty, got %v / %v", got[0], got[1])
	}
	if len(got[2]) != 2 {
		t.Errorf("untiered bucket should have 2 profiles, got %d", len(got[2]))
	}
}

func TestSelectTiers_EmptyInput(t *testing.T) {
	got := SelectTiers(nil, []TierClass{TierFast, TierStrong})
	if len(got) != 3 {
		t.Fatalf("expected 3 buckets (incl untiered), got %d", len(got))
	}
	for i, b := range got {
		if len(b) != 0 {
			t.Errorf("bucket %d should be empty, got %v", i, b)
		}
	}
}

func TestIsValidTier(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"fast", true},
		{"balanced", true},
		{"strong", true},
		{"local", true},
		{"", true},
		{"hyper", false},
		{"FAST", false},
		{"garbage", false},
	}
	for _, c := range cases {
		if got := IsValidTier(c.in); got != c.want {
			t.Errorf("IsValidTier(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCapabilityProfile_TierClass_BackCompatJSON(t *testing.T) {
	// Old profile JSON without tier_class deserialises with empty TierClass.
	oldJSON := []byte(`{"harness":"claude","provider":"anthropic","model":"sonnet","capabilities":{}}`)
	var p CapabilityProfile
	if err := json.Unmarshal(oldJSON, &p); err != nil {
		t.Fatalf("unmarshal old profile: %v", err)
	}
	if p.TierClass != "" {
		t.Errorf("old profile TierClass = %q, want empty", p.TierClass)
	}
	// New profile with tier_class set.
	newJSON := []byte(`{"harness":"opencode","provider":"ollama","model":"q","capabilities":{},"tier_class":"local"}`)
	var p2 CapabilityProfile
	if err := json.Unmarshal(newJSON, &p2); err != nil {
		t.Fatalf("unmarshal new profile: %v", err)
	}
	if p2.TierClass != TierLocal {
		t.Errorf("new profile TierClass = %q, want %q", p2.TierClass, TierLocal)
	}
	// omitempty: marshalling a profile with empty TierClass MUST omit the field.
	out, err := json.Marshal(&CapabilityProfile{Harness: "x", Provider: "y", Model: "z"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(out); got != "" && stringContains(got, "tier_class") {
		t.Errorf("empty TierClass should be omitted; got %s", got)
	}
}

func stringContains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
