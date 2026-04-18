package security

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeField_StripsScriptTags(t *testing.T) {
	input := `<script>alert('xss')</script>`
	output := SanitizeField(input)
	assert.NotContains(t, output, "<script>")
	assert.NotContains(t, output, "alert('xss')")
}

func TestSanitizeField_StripsEventHandlers(t *testing.T) {
	input := `<img src="x" onerror="alert('xss')">`
	output := SanitizeField(input)
	assert.NotContains(t, output, "onerror")
	assert.NotContains(t, output, "alert('xss')")
}

func TestSanitizeField_StripsIframe(t *testing.T) {
	input := `<iframe src="https://evil.com"></iframe>`
	output := SanitizeField(input)
	assert.NotContains(t, output, "<iframe")
}

func TestSanitizeField_StripsJavascriptURL(t *testing.T) {
	input := `[click me](javascript:alert('xss'))`
	output := SanitizeField(input)
	assert.NotContains(t, output, "javascript:")
}

func TestSanitizeField_AllowsSafeMarkdown(t *testing.T) {
	input := "**bold** *italic* `code`"
	output := SanitizeField(input)
	// Safe formatting should be preserved (converted to HTML)
	assert.Contains(t, output, "<strong>")
	assert.Contains(t, output, "<em>")
	assert.Contains(t, output, "<code>")
}

func TestSanitizeField_AllowsSafeLinks(t *testing.T) {
	input := "[safe link](https://example.com)"
	output := SanitizeField(input)
	assert.Contains(t, output, "https://example.com")
	assert.NotContains(t, output, "javascript:")
}

func TestSanitizeField_StripsStyleTag(t *testing.T) {
	input := `<style>body{background:url("javascript:alert(1)")}</style>`
	output := SanitizeField(input)
	assert.NotContains(t, output, "<style")
}

func TestSanitizeField_StripsObjectTag(t *testing.T) {
	input := `<object data="https://evil.com/payload.swf"></object>`
	output := SanitizeField(input)
	assert.NotContains(t, output, "<object")
}

func TestSanitizeField_StripsEmbedTag(t *testing.T) {
	input := `<embed src="data:text/html,<script>alert(1)</script>">`
	output := SanitizeField(input)
	assert.NotContains(t, output, "<embed")
}

func TestSanitizeField_XSSAttackVectors(t *testing.T) {
	attackVectors := []struct {
		name    string
		input   string
		blocked string
	}{
		{
			name:    "svg onload",
			input:   `<svg onload="alert('xss')">`,
			blocked: "onload",
		},
		{
			name:    "body onload",
			input:   `<body onload="document.cookie">content</body>`,
			blocked: "onload",
		},
		{
			name:    "div onmouseover",
			input:   `<div onmouseover="alert(1)">hover me</div>`,
			blocked: "onmouseover",
		},
		{
			name:    "details ontoggle",
			input:   `<details open ontoggle="alert('xss')"><summary>Click</summary></details>`,
			blocked: "ontoggle",
		},
		{
			name:    "base tag",
			input:   `<base href="https://evil.com/">`,
			blocked: "<base",
		},
		{
			name:    "form action",
			input:   `<form action="https://evil.com/steal"><input type="submit"></form>`,
			blocked: "<form",
		},
		{
			name:    "meta refresh",
			input:   `<meta http-equiv="refresh" content="0;url=https://evil.com">`,
			blocked: "<meta",
		},
		{
			name:    "link stylesheet",
			input:   `<link rel="stylesheet" href="https://evil.com/malicious.css">`,
			blocked: "<link",
		},
		{
			name:    "data URI in img",
			input:   `<img src="data:text/html,<script>alert(1)</script>">`,
			blocked: "<script>",
		},
		{
			name:    "HTML comment with conditional",
			input:   `<!--[if IE]><script>alert('xss')</script><![endif]-->`,
			blocked: "<script>",
		},
	}

	for _, tv := range attackVectors {
		t.Run(tv.name, func(t *testing.T) {
			output := SanitizeField(tv.input)
			assert.NotContains(t, output, tv.blocked,
				"should strip %s from input: %s", tv.blocked, tv.input)
		})
	}
}

func TestSanitizeOutput_JSON(t *testing.T) {
	input := map[string]interface{}{
		"title":   "<script>alert('xss')</script>",
		"content": "normal text",
		"nested": map[string]interface{}{
			"html": "<img src=x onerror=alert(1)>",
		},
		"array": []interface{}{
			"<iframe src=evil.com>",
			"normal",
		},
	}

	jsonBytes, err := json.Marshal(input)
	require.NoError(t, err)

	// Parse JSON back to interface{}
	var parsed interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	output := SanitizeOutput(parsed)

	result, ok := output.(map[string]interface{})
	require.True(t, ok, "SanitizeOutput should return map[string]interface{}")

	// Check that dangerous content was stripped
	title := result["title"].(string)
	assert.NotContains(t, title, "<script>")
	assert.NotContains(t, title, "alert('xss')")

	nested := result["nested"].(map[string]interface{})
	html := nested["html"].(string)
	assert.NotContains(t, html, "onerror")

	arr := result["array"].([]interface{})
	assert.NotContains(t, arr[0].(string), "<iframe")
}

func TestSanitizeOutput_PreservesStructure(t *testing.T) {
	input := map[string]interface{}{
		"string": "value",
		"number": 42,
		"bool":   true,
		"null":   nil,
		"array":  []interface{}{1, 2, 3},
		"object": map[string]interface{}{"key": "value"},
	}

	jsonBytes, err := json.Marshal(input)
	require.NoError(t, err)

	// Parse JSON back to interface{}
	var parsed interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	output := SanitizeOutput(parsed)

	result, ok := output.(map[string]interface{})
	require.True(t, ok)

	// String values are sanitized (markdown -> HTML)
	assert.Contains(t, result["string"].(string), "value")
	assert.Equal(t, float64(42), result["number"])
	assert.Equal(t, true, result["bool"])
	assert.Equal(t, nil, result["null"])

	arr := result["array"].([]interface{})
	assert.Len(t, arr, 3)

	obj := result["object"].(map[string]interface{})
	assert.Contains(t, obj["key"].(string), "value")
}

func TestSanitizeText(t *testing.T) {
	input := `<script>alert('xss')</script> normal text`
	output := SanitizeText(input)
	assert.NotContains(t, output, "<script>")
	assert.Contains(t, output, "normal text")
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		contains []string
	}{
		{
			name:     "bold text",
			markdown: "**bold** text",
			contains: []string{"bold", "text"},
		},
		{
			name:     "link",
			markdown: "[link](https://example.com)",
			contains: []string{"link", "https://example.com"},
		},
		{
			name:     "code block",
			markdown: "```go\nfunc main() {}\n```",
			contains: []string{"func main() {}"},
		},
		{
			name:     "image",
			markdown: "![alt](image.png)",
			contains: []string{"[image:", "alt", "image.png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := StripMarkdown(tt.markdown)
			for _, substr := range tt.contains {
				assert.Contains(t, output, substr)
			}
		})
	}
}

func TestSanitizeMarkdownLink_SafeURLs(t *testing.T) {
	safeURLs := []string{
		"https://example.com",
		"http://example.com/path?query=value",
		"https://example.com:8080/path",
	}

	for _, url := range safeURLs {
		t.Run(url, func(t *testing.T) {
			output := SanitizeMarkdownLink(url)
			assert.Equal(t, url, output, "safe URL should not be modified")
		})
	}
}

func TestSanitizeMarkdownLink_DangerousProtocols(t *testing.T) {
	dangerousURLs := []struct {
		input  string
		blocked string
	}{
		{"javascript:alert(1)", "javascript"},
		{"vbscript:msgbox(1)", "vbscript"},
		{"data:text/html,<script>alert(1)</script>", "data:"},
		{"file:///etc/passwd", "file:"},
		{"ftp://example.com", "ftp:"},
		{"mailto:test@example.com", "mailto:"},
	}

	for _, tc := range dangerousURLs {
		t.Run(tc.input, func(t *testing.T) {
			output := SanitizeMarkdownLink(tc.input)
			assert.NotContains(t, output, tc.blocked, "dangerous protocol should be removed")
			assert.Equal(t, "#removed-unsafe-link", output)
		})
	}
}

func TestContainsUnsafeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		unsafe   bool
	}{
		{
			name:   "safe text",
			input:  "normal text",
			unsafe: false,
		},
		{
			name:   "script tag",
			input:  "<script>alert(1)</script>",
			unsafe: true,
		},
		{
			name:   "event handler",
			input:  `<img src=x onerror="alert(1)">`,
			unsafe: true,
		},
		{
			name:   "iframe",
			input:  `<iframe src="evil.com"></iframe>`,
			unsafe: true,
		},
		{
			name:   "safe html",
			input:  "<b>bold</b> and <i>italic</i>",
			unsafe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsUnsafeHTML(tt.input)
			assert.Equal(t, tt.unsafe, result)
		})
	}
}

func TestGetAllowedTags(t *testing.T) {
	tags := GetAllowedTags()

	assert.Contains(t, tags, "b")
	assert.Contains(t, tags, "i")
	assert.Contains(t, tags, "em")
	assert.Contains(t, tags, "strong")
	assert.Contains(t, tags, "p")
	assert.Contains(t, tags, "a")
	assert.Contains(t, tags, "code")
	assert.Contains(t, tags, "pre")

	assert.NotContains(t, tags, "script")
	assert.NotContains(t, tags, "iframe")
	assert.NotContains(t, tags, "object")
}

func TestGetAllowedURLSchemes(t *testing.T) {
	schemes := GetAllowedURLSchemes()

	assert.Contains(t, schemes, "http")
	assert.Contains(t, schemes, "https")

	assert.NotContains(t, schemes, "javascript")
	assert.NotContains(t, schemes, "data")
	assert.NotContains(t, schemes, "file")
}

func TestSanitizeField_MarkdownInjection(t *testing.T) {
	input := "# Heading\n\n" +
		"**Bold** and *italic* text\n\n" +
		"```javascript\n" +
		"console.log(\"code block\");\n" +
		"```\n\n" +
		"<script>alert('xss')</script>\n\n" +
		"[link](javascript:alert(1))\n\n" +
		"Normal paragraph."

	output := SanitizeField(input)

	// Safe markdown should be converted to HTML
	assert.Contains(t, output, "<h1>")
	assert.Contains(t, output, "<strong>")
	assert.Contains(t, output, "<em>")

	// Dangerous content should be stripped
	assert.NotContains(t, output, "<script>")
	assert.NotContains(t, output, "javascript:")

	// Code block should be preserved
	assert.Contains(t, output, "console.log")
}

func TestSanitizeField_HTMLInMarkdown(t *testing.T) {
	input := `<div onclick="alert('xss')">**bold text**</div>`

	output := SanitizeField(input)

	// Bold text should be preserved
	assert.Contains(t, output, "<strong>bold text</strong>")

	// Dangerous HTML should be stripped
	assert.NotContains(t, output, "onclick")
	assert.NotContains(t, output, "alert('xss')")
}

func TestSanitizeOutput_EmptyJSON(t *testing.T) {
	input := map[string]interface{}{}

	jsonBytes, err := json.Marshal(input)
	require.NoError(t, err)

	// Parse JSON back to interface{}
	var parsed interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	output := SanitizeOutput(parsed)

	result, ok := output.(map[string]interface{})
	require.True(t, ok)

	assert.Empty(t, result)
}

func TestSanitizeOutput_NestedStringValues(t *testing.T) {
	input := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"dangerous": "<script>alert(1)</script>",
			},
		},
	}

	jsonBytes, err := json.Marshal(input)
	require.NoError(t, err)

	// Parse JSON back to interface{}
	var parsed interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	output := SanitizeOutput(parsed)

	result, ok := output.(map[string]interface{})
	require.True(t, ok)

	level1 := result["level1"].(map[string]interface{})
	level2 := level1["level2"].(map[string]interface{})
	dangerous := level2["dangerous"].(string)

	assert.NotContains(t, dangerous, "<script>")
}

func TestSanitizeField_AllowsSafeHTML(t *testing.T) {
	input := `<b>bold</b> and <i>italic</i> and <code>code</code>`

	output := SanitizeField(input)

	// These tags should be allowed
	assert.Contains(t, output, "<b>")
	assert.Contains(t, output, "<i>")
	assert.Contains(t, output, "<code>")
}

func TestSanitizeField_AllowsTableTags(t *testing.T) {
	input := `| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |`

	output := SanitizeField(input)

	// Tables should be supported
	assert.Contains(t, output, "<table>") // Markdown tables get converted
}

func TestSanitizeField_AllowsLists(t *testing.T) {
	input := `- Item 1
- Item 2
- Item 3`

	output := SanitizeField(input)

	// Lists should be supported
	assert.Contains(t, output, "<ul>")
	assert.Contains(t, output, "<li>")
}

func TestSanitizeOutput_LargeJSON(t *testing.T) {
	// Build a large JSON object
	largeObj := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeObj[fmt.Sprintf("key%d", i)] = fmt.Sprintf("<script>alert(%d)</script>", i)
	}

	jsonBytes, err := json.Marshal(largeObj)
	require.NoError(t, err)

	// Parse JSON back to interface{}
	var parsed interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	output := SanitizeOutput(parsed)

	result, ok := output.(map[string]interface{})
	require.True(t, ok)

	// All script tags should be stripped
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		val := result[key].(string)
		assert.NotContains(t, val, "<script>", "key %d should have script tag stripped", i)
	}
}
