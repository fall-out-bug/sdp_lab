package main

import (
	"os"
	"path/filepath"
)

func ensureTelegramIntakeFiles(repo string) error {
	corePath := filepath.Join(repo, "internal", "intake", "telegram.go")
	testPath := filepath.Join(repo, "internal", "intake", "telegram_test.go")
	if err := os.MkdirAll(filepath.Dir(corePath), 0o755); err != nil {
		return err
	}
	core := `package intake

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Attachment struct {
	Kind string ` + "`json:\"kind\"`" + `
	ID   string ` + "`json:\"id\"`" + `
}

type IntakeInput struct {
	Command     string       ` + "`json:\"command\"`" + `
	FeatureText string       ` + "`json:\"feature_text\"`" + `
	MessageID   int64        ` + "`json:\"message_id\"`" + `
	ChatID      int64        ` + "`json:\"chat_id\"`" + `
	UserID      int64        ` + "`json:\"user_id\"`" + `
	Username    string       ` + "`json:\"username\"`" + `
	Language    string       ` + "`json:\"language\"`" + `
	Attachments []Attachment ` + "`json:\"attachments\"`" + `
	RawText     string       ` + "`json:\"raw_text\"`" + `
}

type telegramUpdate struct {
	Message       *telegramMessage ` + "`json:\"message\"`" + `
	EditedMessage *telegramMessage ` + "`json:\"edited_message\"`" + `
}

type telegramMessage struct {
	MessageID int64            ` + "`json:\"message_id\"`" + `
	Chat      telegramChat     ` + "`json:\"chat\"`" + `
	From      telegramUser     ` + "`json:\"from\"`" + `
	Text      string           ` + "`json:\"text\"`" + `
	Photo     []telegramPhoto  ` + "`json:\"photo\"`" + `
	Document  *telegramFileRef ` + "`json:\"document\"`" + `
	Voice     *telegramFileRef ` + "`json:\"voice\"`" + `
}

type telegramChat struct {
	ID int64 ` + "`json:\"id\"`" + `
}

type telegramUser struct {
	ID           int64  ` + "`json:\"id\"`" + `
	Username     string ` + "`json:\"username\"`" + `
	LanguageCode string ` + "`json:\"language_code\"`" + `
}

type telegramPhoto struct {
	FileID string ` + "`json:\"file_id\"`" + `
}

type telegramFileRef struct {
	FileID string ` + "`json:\"file_id\"`" + `
}

func NormalizeTelegramUpdate(raw []byte) (IntakeInput, error) {
	var upd telegramUpdate
	if err := json.Unmarshal(raw, &upd); err != nil {
		return IntakeInput{}, err
	}
	msg := upd.Message
	if msg == nil {
		msg = upd.EditedMessage
	}
	if msg == nil {
		return IntakeInput{}, fmt.Errorf("telegram update missing message")
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return IntakeInput{}, fmt.Errorf("telegram message text is empty")
	}

	cmd, payload := parseCommand(text)
	input := IntakeInput{Command: cmd, FeatureText: payload, MessageID: msg.MessageID, ChatID: msg.Chat.ID, UserID: msg.From.ID, Username: msg.From.Username, Language: msg.From.LanguageCode, RawText: text}
	input.Attachments = extractAttachments(*msg)
	return input, nil
}

func parseCommand(text string) (string, string) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return "message", trimmed
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "message", trimmed
	}
	cmd := strings.TrimPrefix(parts[0], "/")
	payload := strings.TrimSpace(strings.TrimPrefix(trimmed, parts[0]))
	if cmd == "feature" && payload != "" {
		return "feature", payload
	}
	return cmd, payload
}

func extractAttachments(msg telegramMessage) []Attachment {
	result := make([]Attachment, 0)
	if len(msg.Photo) > 0 {
		result = append(result, Attachment{Kind: "photo", ID: msg.Photo[len(msg.Photo)-1].FileID})
	}
	if msg.Document != nil {
		result = append(result, Attachment{Kind: "document", ID: msg.Document.FileID})
	}
	if msg.Voice != nil {
		result = append(result, Attachment{Kind: "voice", ID: msg.Voice.FileID})
	}
	return result
}
`
	test := `package intake

import "testing"

func TestNormalizeTelegramFeatureCommand(t *testing.T) {
	raw := []byte(` + "`" + `{"message":{"message_id":42,"chat":{"id":1001},"from":{"id":5001,"username":"alice","language_code":"ru"},"text":"/feature add telegram intake flow","photo":[{"file_id":"p1"},{"file_id":"p2"}]}}` + "`" + `)
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
	raw := []byte(` + "`" + `{"edited_message":{"message_id":7,"chat":{"id":2001},"from":{"id":6001,"username":"bob","language_code":"en"},"text":"hello world","document":{"file_id":"doc1"}}}` + "`" + `)
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
	raw := []byte(` + "`" + `{"message":{"message_id":1,"chat":{"id":1},"from":{"id":1},"text":"   "}}` + "`" + `)
	_, err := NormalizeTelegramUpdate(raw)
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}
`
	if err := os.WriteFile(corePath, []byte(core), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(testPath, []byte(test), 0o644); err != nil {
		return err
	}
	return nil
}
