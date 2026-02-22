package intake

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TelegramAttachment is a telegram-specific file reference (Kind+ID).
type TelegramAttachment struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type IntakeInput struct {
	Command     string       `json:"command"`
	FeatureText string       `json:"feature_text"`
	MessageID   int64        `json:"message_id"`
	ChatID      int64        `json:"chat_id"`
	UserID      int64        `json:"user_id"`
	Username    string       `json:"username"`
	Language    string       `json:"language"`
	Attachments []TelegramAttachment `json:"attachments"`
	RawText     string       `json:"raw_text"`
}

type telegramUpdate struct {
	Message       *telegramMessage `json:"message"`
	EditedMessage *telegramMessage `json:"edited_message"`
}

type telegramMessage struct {
	MessageID int64            `json:"message_id"`
	Chat      telegramChat     `json:"chat"`
	From      telegramUser     `json:"from"`
	Text      string           `json:"text"`
	Photo     []telegramPhoto  `json:"photo"`
	Document  *telegramFileRef `json:"document"`
	Voice     *telegramFileRef `json:"voice"`
}

type telegramChat struct {
	ID int64 `json:"id"`
}

type telegramUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type telegramPhoto struct {
	FileID string `json:"file_id"`
}

type telegramFileRef struct {
	FileID string `json:"file_id"`
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
	input := IntakeInput{
		Command:     cmd,
		FeatureText: payload,
		MessageID:   msg.MessageID,
		ChatID:      msg.Chat.ID,
		UserID:      msg.From.ID,
		Username:    msg.From.Username,
		Language:    msg.From.LanguageCode,
		RawText:     text,
	}
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

func extractAttachments(msg telegramMessage) []TelegramAttachment {
	result := make([]TelegramAttachment, 0)
	if len(msg.Photo) > 0 {
		result = append(result, TelegramAttachment{Kind: "photo", ID: msg.Photo[len(msg.Photo)-1].FileID})
	}
	if msg.Document != nil {
		result = append(result, TelegramAttachment{Kind: "document", ID: msg.Document.FileID})
	}
	if msg.Voice != nil {
		result = append(result, TelegramAttachment{Kind: "voice", ID: msg.Voice.FileID})
	}
	return result
}
