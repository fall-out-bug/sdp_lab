package intake

// Request is the normalized intake request from any source.
type Request struct {
	ProjectID   string       `json:"project_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Source      string       `json:"source"` // telegram, cursor, claude-code, opencode, codex, openclaw
	Priority    int          `json:"priority"`
	Labels      []string     `json:"labels"`
	Attachments []Attachment `json:"attachments"`
	UserID      string       `json:"user_id"`
}

// Attachment is an optional file or reference.
type Attachment struct {
	URL  string `json:"url"`
	Type string `json:"type"`
	Name string `json:"name"`
}
