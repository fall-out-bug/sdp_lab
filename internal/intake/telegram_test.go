package intake

import "testing"

func TestNormalizeTelegramFeatureCommand(t *testing.T) {
	raw := []byte(`{"message":{"message_id":42,"chat":{"id":1001},"from":{"id":5001,"username":"alice","language_code":"ru"},"text":"/feature add telegram intake flow","photo":[{"file_id":"p1"},{"file_id":"p2"}]}}`)
	out, err := NormalizeTelegramUpdate(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.Command != "feature" {
		t.Fatalf("expected feature command, got %s", out.Command)
	}
	if out.FeatureText != "add telegram intake flow" {
		t.Fatalf("unexpected payload: %q", out.FeatureText)
	}
	if out.ChatID != 1001 || out.UserID != 5001 || out.MessageID != 42 {
		t.Fatalf("unexpected ids: %#v", out)
	}
	if len(out.Attachments) != 1 || out.Attachments[0].Kind != "photo" || out.Attachments[0].ID != "p2" {
		t.Fatalf("unexpected attachments: %#v", out.Attachments)
	}
}

func TestNormalizeTelegramEditedMessageFallback(t *testing.T) {
	raw := []byte(`{"edited_message":{"message_id":7,"chat":{"id":2001},"from":{"id":6001,"username":"bob","language_code":"en"},"text":"hello world","document":{"file_id":"doc1"}}}`)
	out, err := NormalizeTelegramUpdate(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.Command != "message" {
		t.Fatalf("expected message command, got %s", out.Command)
	}
	if out.FeatureText != "hello world" {
		t.Fatalf("unexpected payload: %q", out.FeatureText)
	}
	if len(out.Attachments) != 1 || out.Attachments[0].Kind != "document" {
		t.Fatalf("unexpected attachments: %#v", out.Attachments)
	}
}

func TestNormalizeTelegramMissingText(t *testing.T) {
	raw := []byte(`{"message":{"message_id":1,"chat":{"id":1},"from":{"id":1},"text":"   "}}`)
	_, err := NormalizeTelegramUpdate(raw)
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}
