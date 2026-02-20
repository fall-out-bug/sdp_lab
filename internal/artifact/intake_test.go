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

func TestGateSignalsHaveUniqueIDsAndKnownPhases(t *testing.T) {
	knownPhases := map[string]struct{}{}
	for _, req := range PhaseRequirements() {
		knownPhases[req.Phase] = struct{}{}
	}

	seen := map[string]struct{}{}
	for _, signal := range GateSignals() {
		if signal.ID == "" {
			t.Fatalf("gate signal has empty id: %+v", signal)
		}
		if _, ok := seen[signal.ID]; ok {
			t.Fatalf("duplicate gate signal id: %s", signal.ID)
		}
		seen[signal.ID] = struct{}{}
		if signal.Phase == "" {
			t.Fatalf("gate signal %s has empty phase", signal.ID)
		}
		if _, ok := knownPhases[signal.Phase]; !ok {
			t.Fatalf("gate signal %s references unknown phase %s", signal.ID, signal.Phase)
		}
		if signal.Description == "" {
			t.Fatalf("gate signal %s missing description", signal.ID)
		}
	}
}

func TestTransitionPrerequisitesReferenceKnownContracts(t *testing.T) {
	knownPhases := map[string]struct{}{}
	knownClassIDs := map[string]struct{}{}
	knownKeys := map[string]struct{}{}
	knownSignals := map[string]struct{}{}

	for _, req := range PhaseRequirements() {
		knownPhases[req.Phase] = struct{}{}
		for _, classID := range req.RequiredClassIDs {
			knownClassIDs[classID] = struct{}{}
		}
		for _, key := range req.AdditionalProvenanceKeys {
			knownKeys[key] = struct{}{}
		}
	}
	for _, key := range BaseProvenanceKeys() {
		knownKeys[key] = struct{}{}
	}
	for _, class := range Classes() {
		knownClassIDs[class.ID] = struct{}{}
	}
	for _, signal := range GateSignals() {
		knownSignals[signal.ID] = struct{}{}
	}

	for _, transition := range TransitionPrerequisites() {
		if transition.FromPhase == "" || transition.ToPhase == "" {
			t.Fatalf("transition has empty phase edge: %+v", transition)
		}
		if transition.FromPhase == transition.ToPhase {
			t.Fatalf("transition has identical phases: %+v", transition)
		}
		if _, ok := knownPhases[transition.FromPhase]; !ok {
			t.Fatalf("transition references unknown from phase %s", transition.FromPhase)
		}
		if _, ok := knownPhases[transition.ToPhase]; !ok {
			t.Fatalf("transition references unknown to phase %s", transition.ToPhase)
		}
		if len(transition.RequiredGateSignals) == 0 {
			t.Fatalf("transition %s->%s missing gate signals", transition.FromPhase, transition.ToPhase)
		}
		for _, signalID := range transition.RequiredGateSignals {
			if _, ok := knownSignals[signalID]; !ok {
				t.Fatalf("transition %s->%s references unknown gate signal %s", transition.FromPhase, transition.ToPhase, signalID)
			}
		}
		if len(transition.RequiredArtifactClassIDs) == 0 {
			t.Fatalf("transition %s->%s missing required artifact classes", transition.FromPhase, transition.ToPhase)
		}
		for _, classID := range transition.RequiredArtifactClassIDs {
			if _, ok := knownClassIDs[classID]; !ok {
				t.Fatalf("transition %s->%s references unknown class id %s", transition.FromPhase, transition.ToPhase, classID)
			}
		}
		if len(transition.RequiredProvenanceKeys) == 0 {
			t.Fatalf("transition %s->%s missing required provenance keys", transition.FromPhase, transition.ToPhase)
		}
		for _, key := range transition.RequiredProvenanceKeys {
			if _, ok := knownKeys[key]; !ok {
				t.Fatalf("transition %s->%s references unknown provenance key %s", transition.FromPhase, transition.ToPhase, key)
			}
		}
	}
}
