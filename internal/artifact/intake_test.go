package artifact

import "testing"

func TestClassesHaveUniqueIDsAndRetention(t *testing.T) {
	seen := map[string]struct{}{}
	for _, class := range Classes() {
		if class.ID == "" {
			t.Fatalf("class has empty id: %+v", class)
		}
		if _, ok := seen[class.ID]; ok {
			t.Fatalf("duplicate class id: %s", class.ID)
		}
		seen[class.ID] = struct{}{}
		if class.RetentionDays <= 0 {
			t.Fatalf("class %s has non-positive retention: %d", class.ID, class.RetentionDays)
		}
		if len(class.RequiredProvenance) == 0 {
			t.Fatalf("class %s has no provenance requirements", class.ID)
		}
	}
}

func TestPhaseRequirementsReferenceKnownClasses(t *testing.T) {
	known := map[string]struct{}{}
	for _, class := range Classes() {
		known[class.ID] = struct{}{}
	}

	for _, req := range PhaseRequirements() {
		if req.Phase == "" {
			t.Fatalf("phase requirement has empty phase: %+v", req)
		}
		if len(req.RequiredClassIDs) == 0 {
			t.Fatalf("phase %s has no required classes", req.Phase)
		}
		for _, classID := range req.RequiredClassIDs {
			if _, ok := known[classID]; !ok {
				t.Fatalf("phase %s references unknown class %s", req.Phase, classID)
			}
		}
	}
}

func TestBaseProvenanceKeysIncludeHashChainCore(t *testing.T) {
	keys := map[string]struct{}{}
	for _, key := range BaseProvenanceKeys() {
		keys[key] = struct{}{}
	}

	for _, required := range []string{"run_id", "artifact_id", "hash", "hash_prev", "captured_at", "phase"} {
		if _, ok := keys[required]; !ok {
			t.Fatalf("missing required provenance key: %s", required)
		}
	}
}
