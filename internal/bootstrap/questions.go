package bootstrap

// Question represents a single prompt in the interactive bootstrap flow.
// Each question maps to a field in GreenfieldConfig via the Key field.
type Question struct {
	Key     string   // maps to GreenfieldConfig field (JSON tag)
	Prompt  string   // displayed text for the user
	Options []string // valid choices
	Default string   // default value if user provides empty input
}

// DefaultQuestions defines the interactive flow questions in order.
// Each question's Key matches the JSON tag on GreenfieldConfig.
var DefaultQuestions = []Question{
	{
		Key:     "project_type",
		Prompt:  "What type of project are you building?",
		Options: []string{"web-service", "cli", "library", "monorepo"},
		Default: "library",
	},
	{
		Key:     "primary_language",
		Prompt:  "What is the primary programming language?",
		Options: []string{"go", "python", "typescript", "rust", "java"},
		Default: "go",
	},
	{
		Key:     "test_strategy",
		Prompt:  "What testing strategy do you prefer?",
		Options: []string{"unit", "integration", "tdd", "minimal"},
		Default: "tdd",
	},
	{
		Key:     "ci_preference",
		Prompt:  "Which CI system will you use?",
		Options: []string{"github-actions", "gitlab-ci", "none"},
		Default: "github-actions",
	},
	{
		Key:     "deploy_target",
		Prompt:  "What is the deployment target?",
		Options: []string{"docker", "kubernetes", "serverless", "none"},
		Default: "none",
	},
}

// QuestionMap returns DefaultQuestions indexed by Key for quick lookup.
func QuestionMap() map[string]Question {
	m := make(map[string]Question, len(DefaultQuestions))
	for _, q := range DefaultQuestions {
		m[q.Key] = q
	}
	return m
}

// IsValidChoice checks whether value is a valid option for the given key.
func IsValidChoice(key, value string) bool {
	for _, q := range DefaultQuestions {
		if q.Key == key {
			for _, opt := range q.Options {
				if opt == value {
					return true
				}
			}
			return false
		}
	}
	return false
}
