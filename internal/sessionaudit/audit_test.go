package sessionaudit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeJSONL(t *testing.T, dir, name string, records []map[string]any) string {
	t.Helper()
	path := filepath.Join(dir, name+".jsonl")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range records {
		require.NoError(t, enc.Encode(r))
	}
	return path
}

func userMsg(text string) map[string]any {
	return map[string]any{
		"message": map[string]any{
			"role":    "user",
			"content": text,
		},
	}
}

func assistantSkill(skill string) map[string]any {
	return map[string]any{
		"message": map[string]any{
			"role": "assistant",
			"content": []map[string]any{
				{
					"type": "tool_use",
					"name": "Skill",
					"input": map[string]any{"skill": skill},
				},
			},
		},
	}
}

func assistantBash(cmd string) map[string]any {
	return map[string]any{
		"message": map[string]any{
			"role": "assistant",
			"content": []map[string]any{
				{
					"type": "tool_use",
					"name": "Bash",
					"input": map[string]any{"command": cmd},
				},
			},
		},
	}
}

func TestParseSession_nudgeAndCtxLoss(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONL(t, dir, "test-session", []map[string]any{
		userMsg("большое сообщение с реальным содержанием, не nudge"),
		userMsg("ок"),
		userMsg("Continue from where you left off."),
		userMsg("да"),
		assistantSkill("review"),
		assistantSkill("build"),
		assistantBash("bd close sdplab-abc --reason=done"),
	})

	s, err := ParseSession(path)
	require.NoError(t, err)

	assert.Equal(t, 4, s.UserMsgs)
	assert.Equal(t, 3, s.AssistantMsgs)
	assert.Equal(t, 1, s.CtxLossCount)
	assert.Equal(t, 2, len(s.Nudges), "ок + да")
	assert.Equal(t, 1, s.ReviewIter)
	assert.Equal(t, []string{"sdplab-abc"}, s.IssuesClosed)
}

func TestParseSession_skillCalls(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONL(t, dir, "skills-session", []map[string]any{
		userMsg("start"),
		userMsg("next"),
		userMsg("third"),
		assistantSkill("build"),
		assistantSkill("build"),
		assistantSkill("delivery-loop"),
	})

	s, err := ParseSession(path)
	require.NoError(t, err)

	assert.Equal(t, 2, s.SkillCalls["build"])
	assert.Equal(t, 1, s.SkillCalls["delivery-loop"])
}

func TestNudgeRate(t *testing.T) {
	s := &Session{UserMsgs: 10, Nudges: []string{"ок", "да"}, CtxLossCount: 1}
	assert.InDelta(t, 30.0, s.NudgeRate(), 0.01)
}

func TestProductivity(t *testing.T) {
	s := &Session{UserMsgs: 50, IssuesClosed: []string{"a", "b"}}
	assert.InDelta(t, 4.0, s.Productivity(), 0.01)
}

func TestSummarise(t *testing.T) {
	sessions := []*Session{
		{ID: "aaa", UserMsgs: 100, Nudges: []string{"ок"}, CtxLossCount: 2, ReviewIter: 1, IssuesClosed: []string{"x"}, SkillCalls: map[string]int{"build": 3}},
		{ID: "bbb", UserMsgs: 50, Nudges: []string{}, CtxLossCount: 0, ReviewIter: 2, IssuesClosed: []string{"y", "z"}, SkillCalls: map[string]int{"review": 2, "build": 1}},
	}

	agg := Summarise(sessions)

	assert.Equal(t, 2, agg.SessionsAnalyzed)
	assert.Equal(t, 150, agg.TotalUserMsgs)
	assert.Equal(t, 1, agg.TotalNudges)
	assert.Equal(t, 2, agg.TotalCtxLoss)
	assert.Equal(t, 3, agg.TotalReviewIter)
	assert.Equal(t, 3, agg.TotalIssuesClosed)
	assert.Equal(t, "build", agg.TopSkills[0].Skill)
	assert.Equal(t, 4, agg.TopSkills[0].Count)
}
