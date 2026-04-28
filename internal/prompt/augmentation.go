package prompt

import (
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

func ContextSegmentsSection(title string, segments []kernel.ContextSegment) string {
	if len(segments) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, segment := range segments {
		content := strings.TrimSpace(segment.Content)
		if content == "" {
			continue
		}
		b.WriteString(content)
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}
