package control

import "testing"

func TestRecordDelivery(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Ship thin slice", "test")
	if err != nil {
		t.Fatal(err)
	}

	card, err = store.RecordDelivery("openclaw", card.ID, DeliveryStateRolledBack, "staging", "Smoke tests failed after rollout", "deploy:staging-42", []string{"followup:hotfix-99"})
	if err != nil {
		t.Fatalf("RecordDelivery error: %v", err)
	}

	if card.DeliveryState != DeliveryStateRolledBack {
		t.Fatalf("delivery_state = %s, want %s", card.DeliveryState, DeliveryStateRolledBack)
	}
	if card.DeliveryTarget != "staging" {
		t.Fatalf("delivery_target = %s, want staging", card.DeliveryTarget)
	}
	if card.RollbackCount != 1 {
		t.Fatalf("rollback_count = %d, want 1", card.RollbackCount)
	}
	if card.RollbackRef != "deploy:staging-42" {
		t.Fatalf("rollback_ref = %s", card.RollbackRef)
	}
	if len(card.FollowupRefs) != 1 || card.FollowupRefs[0] != "followup:hotfix-99" {
		t.Fatalf("followup_refs = %v", card.FollowupRefs)
	}
}
