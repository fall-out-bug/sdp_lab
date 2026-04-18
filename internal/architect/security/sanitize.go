package security

import (
	"strings"
	"sync"

	md "github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
)

var (
	// htmlPolicy is the bluemonday policy for HTML sanitization.
	// It allows only safe tags and attributes.
	htmlPolicy *bluemonday.Policy
	once       sync.Once
)

// initHTMLPolicy initializes the HTML sanitization policy.
func initHTMLPolicy() {
	once.Do(func() {
		htmlPolicy = bluemonday.NewPolicy()

		// Allow basic formatting tags
		htmlPolicy.AllowElements("b", "i", "em", "strong", "p", "br")
		htmlPolicy.AllowElements("ul", "ol", "li")
		htmlPolicy.AllowElements("code", "pre")

		// Allow links with restrictions
		htmlPolicy.AllowElements("a")
		htmlPolicy.AllowAttrs("href").OnElements("a")
		htmlPolicy.AllowURLSchemes("http", "https")

		// Allow standard heading tags
		htmlPolicy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")

		// Allow blockquote
		htmlPolicy.AllowElements("blockquote")

		// Allow tables
		htmlPolicy.AllowElements("table", "thead", "tbody", "tr", "th", "td")

		// Allow div and span for structure
		htmlPolicy.AllowElements("div", "span")
	})
}

// SanitizeField sanitizes a single markdown field by converting to HTML
// and applying bluemonday policy. This prevents XSS attacks.
//
// Process:
// 1. Parse markdown with FlagsNone (no raw HTML passthrough)
// 2. Convert to HTML
// 3. Apply bluemonday policy to strip dangerous elements
func SanitizeField(markdown string) string {
	initHTMLPolicy()

	// Parse markdown with strict flags (no raw HTML)
	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)
	opts := html.RendererOptions{Flags: html.FlagsNone}
	renderer := html.NewRenderer(opts)

	htmlBytes := md.ToHTML([]byte(markdown), p, renderer)

	// Apply bluemonday policy
	return string(htmlPolicy.SanitizeBytes(htmlBytes))
}

// SanitizeOutput recursively sanitizes all string values in JSON data.
// It parses JSON, walks the structure, and sanitizes each string value
// using SanitizeField. This is useful for LLM output that may contain
// markdown/HTML.
//
// Note: This function works with parsed JSON (interface{}), not raw JSON bytes.
// Use architect.SanitizeOutput for the raw JSON bytes version.
func SanitizeOutput(parsed interface{}) interface{} {
	switch v := parsed.(type) {
	case string:
		return SanitizeField(v)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = SanitizeOutput(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = SanitizeOutput(val)
		}
		return result
	default:
		return v
	}
}

// SanitizeText sanitizes plain text by removing HTML tags and dangerous content.
func SanitizeText(text string) string {
	initHTMLPolicy()
	return htmlPolicy.Sanitize(text)
}

// StripMarkdown removes markdown syntax, leaving plain text.
func StripMarkdown(markdown string) string {
	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(markdown))

	var result strings.Builder
	walkText(doc, &result)
	return result.String()
}

// walkText recursively extracts text from markdown AST.
func walkText(node ast.Node, result *strings.Builder) {
	if node == nil {
		return
	}

	switch v := node.(type) {
	case *ast.Text:
		result.Write(v.Literal)
	case *ast.Code:
		result.WriteString(string(v.Literal))
	case *ast.CodeBlock:
		result.Write(v.Literal)
	case *ast.Link:
		// Add link text and URL
		ast.WalkFunc(v, func(node ast.Node, entering bool) ast.WalkStatus {
			if entering && node.AsLeaf() != nil {
				walkText(node, result)
			}
			return ast.GoToNext
		})
		result.WriteString(" (")
		result.WriteString(string(v.Destination))
		result.WriteString(")")
	default:
		// Recursively process children
		ast.WalkFunc(v, func(node ast.Node, entering bool) ast.WalkStatus {
			if entering {
				walkText(node, result)
			}
			return ast.GoToNext
		})
	}

	// Add appropriate whitespace
	switch node.(type) {
	case *ast.Paragraph, *ast.Heading, *ast.ListItem:
		result.WriteString("\n")
	}
}

// SanitizeMarkdownLink sanitizes a URL for use in markdown links.
// It ensures the URL uses a safe protocol (http/https).
func SanitizeMarkdownLink(url string) string {
	initHTMLPolicy()

	// Check if URL has a dangerous protocol
	lowerURL := strings.ToLower(url)
	dangerousPrefixes := []string{
		"javascript:",
		"vbscript:",
		"data:",
		"file:",
		"ftp:",
		"mailto:",
	}

	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(lowerURL, prefix) {
			return "#removed-unsafe-link"
		}
	}

	// Return safe URL as-is
	return url
}

// ContainsUnsafeHTML checks if text contains potentially unsafe HTML.
func ContainsUnsafeHTML(text string) bool {
	initHTMLPolicy()

	// Sanitize and compare
	sanitized := htmlPolicy.Sanitize(text)
	return sanitized != text
}

// GetAllowedTags returns a list of allowed HTML tags.
func GetAllowedTags() []string {
	initHTMLPolicy()

	return []string{
		"b", "i", "em", "strong", "p", "br",
		"ul", "ol", "li", "code", "pre",
		"a", "h1", "h2", "h3", "h4", "h5", "h6",
		"blockquote", "table", "thead", "tbody", "tr", "th", "td",
		"div", "span",
	}
}

// GetAllowedURLSchemes returns a list of allowed URL schemes.
func GetAllowedURLSchemes() []string {
	initHTMLPolicy()
	return []string{"http", "https"}
}
