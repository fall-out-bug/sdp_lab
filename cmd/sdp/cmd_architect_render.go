package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func runArchitectRender(args []string) {
	fs := flag.NewFlagSet("architect render", flag.ExitOnError)
	outputFlag := fs.String("o", "", "output HTML path (default: same name .html)")
	fs.StringVar(outputFlag, "output", "", "output HTML path (default: same name .html)")
	openFlag := fs.Bool("open", false, "open in browser after rendering")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("flag parse error: %v", err)
	}

	mdPath := fs.Arg(0)
	if mdPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sdp architect render [flags] <report.md>")
		fs.PrintDefaults()
		os.Exit(2)
	}

	absPath, err := filepath.Abs(mdPath)
	if err != nil {
		log.Fatalf("failed to resolve path: %v", err)
	}

	mdBytes, err := os.ReadFile(absPath)
	if err != nil {
		log.Fatalf("failed to read %s: %v", mdPath, err)
	}
	mdText := string(mdBytes)

	outPath := *outputFlag
	if outPath == "" {
		ext := filepath.Ext(absPath)
		outPath = absPath[:len(absPath)-len(ext)] + ".html"
	}

	result := renderReport(mdText)

	if err := os.WriteFile(outPath, []byte(result.html), 0644); err != nil {
		log.Fatalf("failed to write %s: %v", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "✓ Отрендерено: %s\n", outPath)
	fmt.Fprintf(os.Stderr, "  Секций: %d\n", result.sectionCount)
	fmt.Fprintf(os.Stderr, "  Диаграмм Mermaid: %d\n", result.mermaidCount)

	if *openFlag {
		openBrowser("file://" + outPath)
	}
}

// --- rendering types ---

type renderResult struct {
	html         string
	sectionCount int
	mermaidCount int
}

type section struct {
	title   string
	id      string
	content []string
}

// --- main render ---

func renderReport(mdText string) renderResult {
	title := "Architecture Report"
	if m := reH1.FindStringSubmatch(mdText); m != nil {
		title = strings.TrimSpace(m[1])
	}

	meta := ""
	if m := reMeta.FindStringSubmatch(mdText); m != nil {
		meta = strings.TrimSpace(m[1])
	}

	processed, mermaidBlocks := extractMermaidBlocks(mdText)
	sections := parseSections(processed)

	// Nav
	var navHTML strings.Builder
	for _, sec := range sections {
		if sec.title == "" {
			continue
		}
		num := ""
		display := sec.title
		if m := reSecNum.FindStringSubmatch(sec.title); m != nil {
			num = m[1]
			display = strings.TrimSpace(m[2])
		}
		if len(display) > 40 {
			display = display[:38] + "…"
		}
		fmt.Fprintf(&navHTML, `<a class="nav-item" href="#%s"><span class="nav-number">%s.</span> %s</a>`+"\n",
			sec.id, num, html.EscapeString(display))
	}

	// Content
	var contentHTML strings.Builder
	mermaidTotal := 0
	for _, sec := range sections {
		fmt.Fprintf(&contentHTML, `<div class="section" id="%s">`, sec.id)
		if sec.title != "" {
			fmt.Fprintf(&contentHTML, "<h2>%s</h2>\n", mdInline(sec.title))
		}
		sectionHTML := mdToHTML(sec.content, mermaidBlocks, sec.title)
		contentHTML.WriteString(sectionHTML)
		contentHTML.WriteString("</div>\n")
		mermaidTotal += strings.Count(sectionHTML, "mermaid-wrapper")
	}

	sidebarTitle := title
	if idx := strings.Index(sidebarTitle, " — "); idx > 0 {
		sidebarTitle = sidebarTitle[:idx]
	}
	if idx := strings.Index(sidebarTitle, " — "); idx > 0 {
		sidebarTitle = sidebarTitle[:idx]
	}

	metaDisplay := strings.ReplaceAll(meta, "|", "&bull;")

	finalHTML := htmlTemplate
	finalHTML = strings.ReplaceAll(finalHTML, "{{TITLE}}", html.EscapeString(title))
	finalHTML = strings.ReplaceAll(finalHTML, "{{SIDEBAR_TITLE}}", html.EscapeString(sidebarTitle))
	finalHTML = strings.ReplaceAll(finalHTML, "{{SIDEBAR_META}}", metaDisplay)
	finalHTML = strings.ReplaceAll(finalHTML, "{{NAV_ITEMS}}", navHTML.String())
	finalHTML = strings.ReplaceAll(finalHTML, "{{CONTENT}}", contentHTML.String())

	return renderResult{
		html:         finalHTML,
		sectionCount: len(sections),
		mermaidCount: len(mermaidBlocks),
	}
}

// --- markdown parsing ---

var (
	reH1     = regexp.MustCompile(`(?m)^# (.+)$`)
	reH2     = regexp.MustCompile(`^## (.+)$`)
	reH3     = regexp.MustCompile(`^### (.+)$`)
	reH4     = regexp.MustCompile(`^#### (.+)$`)
	reMeta   = regexp.MustCompile(`> \*\*(?:Date|Дата)\*\*: (.+?)(?:\n|$)`)
	reSecNum = regexp.MustCompile(`^(\d+)\.?\s+(.+)$`)
	reHR     = regexp.MustCompile(`^-{3,}$`)
	reUL     = regexp.MustCompile(`^[-*]\s+(.+)$`)
	reOL     = regexp.MustCompile(`^\d+\.\s+(.+)$`)
	reMM     = regexp.MustCompile(`^%%MERMAID_(\d+)%%$`)

	// inline patterns
	reEvExtracted  = regexp.MustCompile(`\[(?:EXTRACTED|ИЗВЛЕЧЕНО)[^\]]*\]`)
	reEvInferred   = regexp.MustCompile(`\[(?:INFERRED|ВЫВЕДЕНО)[^\]]*\]`)
	reEvAmbiguous  = regexp.MustCompile(`\[(?:AMBIGUOUS|НЕОДНОЗНАЧНО)[^\]]*\]`)
	reBold         = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic       = regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*(?:[^*]|$)`)
	reInlineCode   = regexp.MustCompile("`([^`]+)`")
	reLink         = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func parseSections(mdText string) []section {
	var sections []section
	var current *section
	lines := strings.Split(mdText, "\n")

	for _, line := range lines {
		if m := reH2.FindStringSubmatch(line); m != nil {
			if current != nil {
				sections = append(sections, *current)
			}
			current = &section{
				title:   strings.TrimSpace(m[1]),
				id:      fmt.Sprintf("section-%d", len(sections)),
				content: nil,
			}
		} else if current != nil {
			current.content = append(current.content, line)
		} else {
			current = &section{id: "header"}
			current.content = append(current.content, line)
		}
	}
	if current != nil {
		sections = append(sections, *current)
	}
	return sections
}

func extractMermaidBlocks(text string) (string, []string) {
	var blocks []string
	re := regexp.MustCompile("(?s)```mermaid\\s*\\n(.*?)\\n```")
	out := re.ReplaceAllStringFunc(text, func(match string) string {
		m := re.FindStringSubmatch(match)
		idx := len(blocks)
		blocks = append(blocks, sanitizeMermaid(strings.TrimSpace(m[1])))
		return fmt.Sprintf("%%%%MERMAID_%d%%%%", idx)
	})
	return out, blocks
}

// sanitizeMermaid fixes common issues that break mermaid rendering:
// - <br/> and <br> tags → \n (mermaid line breaks)
// - emoji in node labels → stripped (breaks parser)
func sanitizeMermaid(code string) string {
	// Replace HTML line breaks with mermaid-compatible \n
	code = strings.ReplaceAll(code, "<br/>", "\\n")
	code = strings.ReplaceAll(code, "<br>", "\\n")
	code = strings.ReplaceAll(code, "<br />", "\\n")

	// Strip emoji that break mermaid parsing (common severity markers)
	reEmoji := regexp.MustCompile(`[\x{1F534}\x{1F7E1}\x{1F7E2}\x{1F7E0}\x{1F535}\x{2B55}\x{26A0}\x{2705}\x{274C}]`)
	code = reEmoji.ReplaceAllString(code, "")

	return code
}

func isFlowBlock(lines []string) bool {
	arrows := 0
	for _, l := range lines {
		if strings.Contains(l, "→") || strings.Contains(l, "->") {
			arrows++
		}
	}
	return arrows >= 3
}

func renderFlowBlock(lines []string) string {
	var b strings.Builder
	b.WriteString(`<div class="flow-block">`)
	for _, line := range lines {
		esc := html.EscapeString(line)
		esc = strings.ReplaceAll(esc, "→", `<span class="flow-arrow">→</span>`)
		// Color parenthetical details
		esc = regexp.MustCompile(`\(([^)]+)\)`).ReplaceAllString(esc, `<span class="flow-detail">($1)</span>`)
		b.WriteString(esc)
		b.WriteByte('\n')
	}
	b.WriteString("</div>")
	return b.String()
}

func detectC4Level(code, sectionTitle string) int {
	// Canonical C4 diagram types take priority
	if strings.HasPrefix(code, "C4Context") {
		return 1
	}
	if strings.HasPrefix(code, "C4Container") {
		return 2
	}
	if strings.HasPrefix(code, "C4Component") {
		return 3
	}
	if strings.HasPrefix(code, "C4Deployment") {
		return 4
	}

	// Fallback: detect from section title
	t := strings.ToLower(sectionTitle)
	switch {
	case strings.Contains(t, "l1") || strings.Contains(t, "ландшафт") || strings.Contains(t, "landscape") || strings.Contains(t, "контекст"):
		return 1
	case strings.Contains(t, "l2") || strings.Contains(t, "карта модул") || strings.Contains(t, "module map") || strings.Contains(t, "контейнер"):
		return 2
	case strings.Contains(t, "l3") || strings.Contains(t, "компонент") || strings.Contains(t, "component"):
		return 3
	}
	if strings.Count(code, "subgraph") >= 4 {
		return 2
	}
	return 0
}

func mdInline(text string) string {
	// Evidence tags
	text = reEvExtracted.ReplaceAllStringFunc(text, func(s string) string {
		return `<span class="tag tag-extracted">` + html.EscapeString(s) + `</span>`
	})
	text = reEvInferred.ReplaceAllStringFunc(text, func(s string) string {
		return `<span class="tag tag-inferred">` + html.EscapeString(s) + `</span>`
	})
	text = reEvAmbiguous.ReplaceAllStringFunc(text, func(s string) string {
		return `<span class="tag tag-ambiguous">` + html.EscapeString(s) + `</span>`
	})

	// Severity
	text = strings.ReplaceAll(text, "🔴", `<span class="severity-red">⬤</span>`)
	text = strings.ReplaceAll(text, "🟡", `<span class="severity-yellow">⬤</span>`)
	text = strings.ReplaceAll(text, "🟢", `<span class="severity-green">⬤</span>`)

	// Bold, italic, code, links
	text = reBold.ReplaceAllString(text, `<strong>$1</strong>`)
	text = reInlineCode.ReplaceAllString(text, `<code>$1</code>`)
	text = reLink.ReplaceAllString(text, `<a href="$2">$1</a>`)

	return text
}

func mdToHTML(lines []string, mermaidBlocks []string, sectionTitle string) string {
	var out strings.Builder
	inTable, inList, inCode, inBQ := false, false, false, false
	var listType, codeLang string
	var codeLines, bqLines []string

	flushList := func() {
		if inList {
			out.WriteString("</" + listType + ">\n")
			inList = false
		}
	}
	flushBQ := func() {
		if inBQ {
			out.WriteString("<blockquote>" + mdInline(strings.Join(bqLines, " ")) + "</blockquote>\n")
			bqLines = nil
			inBQ = false
		}
	}

	for _, line := range lines {
		stripped := strings.TrimSpace(line)

		// Code fence
		if strings.HasPrefix(stripped, "```") && !inCode {
			flushList()
			flushBQ()
			inCode = true
			codeLang = strings.TrimPrefix(stripped, "```")
			codeLines = nil
			continue
		}
		if stripped == "```" && inCode {
			inCode = false
			if isFlowBlock(codeLines) {
				out.WriteString(renderFlowBlock(codeLines))
			} else {
				codeText := html.EscapeString(strings.Join(codeLines, "\n"))
				fmt.Fprintf(&out, "<pre><code class=\"language-%s\">%s</code></pre>\n", codeLang, codeText)
			}
			continue
		}
		if inCode {
			codeLines = append(codeLines, line)
			continue
		}

		// Mermaid placeholder
		if m := reMM.FindStringSubmatch(stripped); m != nil {
			flushList()
			flushBQ()
			idx := 0
			fmt.Sscanf(m[1], "%d", &idx)
			code := ""
			if idx < len(mermaidBlocks) {
				code = mermaidBlocks[idx]
			}
			escaped := html.EscapeString(code)
			escapedAttr := strings.ReplaceAll(escaped, "\"", "&quot;")

			c4Level := detectC4Level(code, sectionTitle)
			badge := ""
			if c4Level > 0 {
				badge = fmt.Sprintf(`<span class="c4-badge c4-l%d">C4 Level %d</span>`, c4Level, c4Level)
			}

			fmt.Fprintf(&out, `<div class="mermaid-wrapper">
  <div class="mermaid-toolbar">%s
    <button class="mermaid-btn mz-in" title="Приблизить">+</button>
    <button class="mermaid-btn mz-out" title="Отдалить">&minus;</button>
    <button class="mermaid-btn mz-reset" title="Сброс">&#8634;</button>
  </div>
  <div class="mermaid-viewport">
    <div class="mermaid-inner">
      <div class="mermaid" data-mermaid="%s">%s</div>
    </div>
  </div>
  <div class="mermaid-zoom-hint">🖱 Колесо = зум, перетаскивание = пан</div>
</div>`+"\n", badge, escapedAttr, escaped)
			continue
		}

		// Blockquote
		if strings.HasPrefix(stripped, "> ") {
			flushList()
			content := stripped[2:]
			if !inBQ {
				inBQ = true
				bqLines = []string{content}
			} else {
				bqLines = append(bqLines, content)
			}
			continue
		} else if inBQ {
			flushBQ()
		}

		// Headings
		if m := reH3.FindStringSubmatch(stripped); m != nil {
			flushList()
			fmt.Fprintf(&out, "<h3>%s</h3>\n", mdInline(m[1]))
			continue
		}
		if m := reH4.FindStringSubmatch(stripped); m != nil {
			flushList()
			fmt.Fprintf(&out, "<h4>%s</h4>\n", mdInline(m[1]))
			continue
		}

		// HR
		if reHR.MatchString(stripped) {
			flushList()
			out.WriteString("<hr>\n")
			continue
		}

		// Table
		if strings.HasPrefix(stripped, "|") && strings.Contains(stripped, "|") {
			cells := splitTableRow(stripped)
			if isTableSeparator(cells) {
				continue
			}
			if !inTable {
				flushList()
				inTable = true
				out.WriteString("<table><thead><tr>")
				for _, c := range cells {
					fmt.Fprintf(&out, "<th>%s</th>", mdInline(c))
				}
				out.WriteString("</tr></thead><tbody>\n")
			} else {
				out.WriteString("<tr>")
				for _, c := range cells {
					fmt.Fprintf(&out, "<td>%s</td>", mdInline(c))
				}
				out.WriteString("</tr>\n")
			}
			continue
		} else if inTable {
			out.WriteString("</tbody></table>\n")
			inTable = false
		}

		// Lists
		if m := reUL.FindStringSubmatch(stripped); m != nil {
			flushBQ()
			if !inList || listType != "ul" {
				flushList()
				inList = true
				listType = "ul"
				out.WriteString("<ul>\n")
			}
			fmt.Fprintf(&out, "<li>%s</li>\n", mdInline(m[1]))
			continue
		}
		if m := reOL.FindStringSubmatch(stripped); m != nil {
			flushBQ()
			if !inList || listType != "ol" {
				flushList()
				inList = true
				listType = "ol"
				out.WriteString("<ol>\n")
			}
			fmt.Fprintf(&out, "<li>%s</li>\n", mdInline(m[1]))
			continue
		}

		if inList && stripped == "" {
			flushList()
			continue
		}

		if stripped == "" {
			continue
		}

		// Paragraph
		flushList()
		fmt.Fprintf(&out, "<p>%s</p>\n", mdInline(stripped))
	}

	flushList()
	flushBQ()
	if inTable {
		out.WriteString("</tbody></table>\n")
	}
	return out.String()
}

func splitTableRow(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	parts := strings.Split(row, "|")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func isTableSeparator(cells []string) bool {
	sep := regexp.MustCompile(`^[-:]+$`)
	for _, c := range cells {
		if !sep.MatchString(c) {
			return false
		}
	}
	return true
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		fmt.Fprintf(os.Stderr, "Open in browser: %s\n", url)
		return
	}
	_ = cmd.Start()
}

// --- HTML template ---
// Faust design system, Mermaid.js, pan+zoom, scroll-spy sidebar

const htmlTemplate = `<!DOCTYPE html>
<html lang="ru" data-theme="dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{TITLE}}</title>
<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
:root {
  --accent: #4760F3; --accent-hover: #8082F4; --accent-lavender: #8082F4;
  --accent-light: #A8B5FF; --surface-tint: rgba(71,96,243,0.12);
  --danger: #FF8080; --warning: #fbbf24; --success: #34d399;
}
[data-theme="dark"] {
  --bg:#0f0f0f; --bg-elevated:#1a1a1a; --bg-hover:#222; --bg-active:#2a2a2a;
  --text:#e8e8e8; --text-secondary:#8B93A7; --text-muted:#555;
  --border:#2a2a2a; --border-strong:#3a3a3a;
  --shadow-sm:0 1px 2px rgba(71,96,243,.12); --shadow-md:0 6px 18px rgba(71,96,243,.16);
  --shadow-lg:0 12px 32px rgba(71,96,243,.2); --code-bg:#161622;
  --flow-bg:#12121e; --flow-arrow:#A8B5FF; --flow-text:#e8e8e8; --flow-dim:#8B93A7;
}
[data-theme="light"] {
  --bg:#fff; --bg-elevated:#f9f9f9; --bg-hover:#f3f3f3; --bg-active:#ebebeb;
  --text:#1a1a1a; --text-secondary:#6C778E; --text-muted:#aaa;
  --border:#e5e5e5; --border-strong:#d0d0d0;
  --shadow-sm:0 1px 2px rgba(71,96,243,.06); --shadow-md:0 6px 18px rgba(71,96,243,.08);
  --shadow-lg:0 12px 32px rgba(71,96,243,.12); --code-bg:#f5f5ff;
  --flow-bg:#f0f0ff; --flow-arrow:#4760F3; --flow-text:#1a1a1a; --flow-dim:#6C778E;
}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Inter',system-ui,sans-serif;background:var(--bg);color:var(--text);line-height:1.6;font-weight:300;font-size:15px}
.page-wrapper{display:flex;min-height:100vh}
.sidebar{width:280px;background:var(--bg-elevated);border-right:1px solid var(--border);padding:24px 16px;position:fixed;top:0;left:0;bottom:0;overflow-y:auto;z-index:10}
.sidebar-header{padding:8px 12px 24px;border-bottom:1px solid var(--border);margin-bottom:16px}
.sidebar-header h2{font-size:16px;font-weight:600;color:var(--accent);letter-spacing:-.01em}
.sidebar-header .meta{font-size:12px;color:var(--text-secondary);margin-top:4px}
.nav-item{display:block;padding:8px 12px;border-radius:8px;color:var(--text-secondary);text-decoration:none;font-size:14px;font-weight:400;transition:background .15s,color .15s;cursor:pointer;margin-bottom:2px;border-left:3px solid transparent}
.nav-item:hover{background:var(--bg-hover);color:var(--text)}
.nav-item.active{background:var(--surface-tint);color:var(--accent);font-weight:500;border-left-color:var(--accent)}
.nav-item .nav-number{display:inline-block;width:24px;font-size:12px;color:var(--text-muted);font-weight:500}
.main-content{margin-left:280px;max-width:920px;padding:48px 64px;flex:1}
.theme-toggle{position:fixed;top:16px;right:16px;z-index:100;background:var(--bg-elevated);border:1px solid var(--border);border-radius:9999px;padding:8px 16px;cursor:pointer;color:var(--text-secondary);font-size:13px;font-family:inherit;box-shadow:var(--shadow-sm)}
.theme-toggle:hover{background:var(--bg-hover);color:var(--text)}
h1{font-size:28px;font-weight:700;line-height:1.1;letter-spacing:-.02em;margin:48px 0 24px}h1:first-child{margin-top:0}
h2{font-size:22px;font-weight:600;margin:48px 0 16px;padding-bottom:12px;border-bottom:1px solid var(--border)}
h3{font-size:17px;font-weight:500;margin:32px 0 12px}
h4{font-size:15px;font-weight:500;margin:24px 0 8px;color:var(--accent-lavender)}
p{margin:0 0 16px}
a{color:var(--accent);text-decoration:none}a:hover{color:var(--accent-hover);text-decoration:underline}
strong{font-weight:600}em{font-style:italic;color:var(--text-secondary)}
blockquote{border-left:3px solid var(--accent);padding:12px 20px;margin:16px 0;background:var(--surface-tint);border-radius:0 8px 8px 0;font-size:14px;color:var(--text-secondary)}
hr{border:none;border-top:1px solid var(--border);margin:32px 0}
code{font-family:'SF Mono','Fira Code',ui-monospace,monospace;background:var(--code-bg);padding:2px 6px;border-radius:4px;font-size:13px;color:var(--accent-light);border:1px solid var(--border)}
pre{background:var(--code-bg);border:1px solid var(--border);border-radius:12px;padding:20px 24px;overflow-x:auto;margin:16px 0;box-shadow:var(--shadow-sm)}
pre code{background:none;padding:0;border:none;font-size:13px;line-height:1.5;color:var(--text)}
.flow-block{background:var(--flow-bg);border:1px solid var(--border);border-radius:12px;padding:24px 28px;margin:16px 0;box-shadow:var(--shadow-sm);font-family:'SF Mono',ui-monospace,monospace;font-size:13px;line-height:2;white-space:pre-wrap;overflow-x:auto}
.flow-block .flow-arrow{color:var(--flow-arrow);font-weight:600}
.flow-block .flow-detail{color:var(--flow-dim);font-size:12px}
table{width:100%;border-collapse:separate;border-spacing:0;margin:16px 0;border-radius:12px;overflow:hidden;border:1px solid var(--border);box-shadow:var(--shadow-sm)}
thead{background:var(--bg-elevated)}
th{padding:12px 16px;text-align:left;font-weight:500;font-size:13px;color:var(--text-secondary);text-transform:uppercase;letter-spacing:.05em;border-bottom:1px solid var(--border)}
td{padding:12px 16px;font-size:14px;border-bottom:1px solid var(--border);vertical-align:top}
tr:last-child td{border-bottom:none}tr:hover td{background:var(--bg-hover)}
ul,ol{padding-left:24px;margin:8px 0 16px}li{margin:4px 0;line-height:1.6}
.tag{display:inline-block;padding:2px 8px;border-radius:9999px;font-size:11px;font-weight:500;letter-spacing:.02em}
.tag-extracted{background:rgba(16,185,129,.15);color:var(--success)}
.tag-inferred{background:rgba(251,191,36,.15);color:var(--warning)}
.tag-ambiguous{background:rgba(255,128,128,.15);color:var(--danger)}
.mermaid-wrapper{background:var(--bg-elevated);border:1px solid var(--border);border-radius:16px;margin:20px 0;box-shadow:var(--shadow-md);position:relative;overflow:hidden}
.mermaid-viewport{width:100%;min-height:200px;overflow:hidden;cursor:grab;position:relative}
.mermaid-viewport:active{cursor:grabbing}
.mermaid-inner{transform-origin:0 0;display:inline-block;padding:24px;min-width:100%;text-align:center}
.mermaid svg{max-width:none!important;height:auto}
.mermaid-toolbar{position:absolute;top:12px;right:12px;display:flex;gap:4px;z-index:5}
.mermaid-btn{background:rgba(15,15,15,.8);backdrop-filter:blur(8px);border:1px solid var(--border);border-radius:6px;padding:4px 10px;color:var(--text-secondary);font-size:14px;cursor:pointer;font-family:inherit;line-height:1}
.mermaid-btn:hover{background:var(--bg-hover);color:var(--text)}
.mermaid-zoom-hint{position:absolute;bottom:8px;left:12px;font-size:11px;color:var(--text-muted);pointer-events:none}
.section{scroll-margin-top:24px}
.severity-red{color:var(--danger)}.severity-yellow{color:var(--warning)}.severity-green{color:var(--success)}
.c4-badge{display:inline-block;padding:2px 10px;border-radius:9999px;font-size:11px;font-weight:600;letter-spacing:.03em;margin-right:8px;vertical-align:middle}
.c4-l1{background:rgba(71,96,243,.2);color:var(--accent)}
.c4-l2{background:rgba(128,130,244,.2);color:var(--accent-lavender)}
.c4-l3{background:rgba(168,181,255,.2);color:var(--accent-light)}
.c4-l4{background:rgba(52,211,153,.2);color:var(--success)}
@media print{.sidebar,.theme-toggle,.mermaid-toolbar,.mermaid-zoom-hint{display:none}.main-content{margin-left:0;max-width:100%;padding:24px}}
@media(max-width:768px){.sidebar{transform:translateX(-100%);width:260px}.sidebar.open{transform:translateX(0)}.main-content{margin-left:0;padding:24px 20px}.mobile-menu{display:block;position:fixed;top:16px;left:16px;z-index:100;background:var(--bg-elevated);border:1px solid var(--border);border-radius:8px;padding:8px 12px;cursor:pointer;color:var(--text);font-size:18px}}
@media(min-width:769px){.mobile-menu{display:none}}
::-webkit-scrollbar{width:6px}::-webkit-scrollbar-track{background:transparent}::-webkit-scrollbar-thumb{background:var(--border-strong);border-radius:3px}
</style>
</head>
<body>
<button class="mobile-menu" onclick="document.querySelector('.sidebar').classList.toggle('open')">&#9776;</button>
<button class="theme-toggle" onclick="toggleTheme()"><span id="ti">☾</span> <span id="tt">Тема</span></button>
<div class="page-wrapper">
  <nav class="sidebar" id="sidebar">
    <div class="sidebar-header">
      <h2>{{SIDEBAR_TITLE}}</h2>
      <div class="meta">{{SIDEBAR_META}}</div>
    </div>
    <div id="nav-items">{{NAV_ITEMS}}</div>
  </nav>
  <main class="main-content" id="content">{{CONTENT}}</main>
</div>
<script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
<script>
function toggleTheme(){
  var h=document.documentElement,n=h.getAttribute('data-theme')==='dark'?'light':'dark';
  h.setAttribute('data-theme',n);
  document.getElementById('ti').textContent=n==='dark'?'☾':'☀';
  initMermaid(n);
}
function initMermaid(theme){
  mermaid.initialize({startOnLoad:false,securityLevel:'loose',theme:theme==='dark'?'dark':'default',
    themeVariables:theme==='dark'?{primaryColor:'#4760F3',primaryTextColor:'#e8e8e8',primaryBorderColor:'#3a3a3a',lineColor:'#8B93A7',secondaryColor:'#1a1a1a',tertiaryColor:'#222',background:'#1a1a1a',mainBkg:'#1a1a1a',nodeBorder:'#3a3a3a',clusterBkg:'#161622',clusterBorder:'#2a2a2a',titleColor:'#e8e8e8',edgeLabelBackground:'#1a1a1a',nodeTextColor:'#e8e8e8'}:{primaryColor:'#4760F3',primaryTextColor:'#1a1a1a',lineColor:'#6C778E'},
    flowchart:{curve:'basis',padding:20,nodeSpacing:30,rankSpacing:50,htmlLabels:true,wrappingWidth:200},
    c4:{personFontSize:14,personFontWeight:'500',c4ShapeMargin:40,c4ShapePadding:16,
      wrap:true,wrapPadding:10,c4BoundaryInRow:3},
    fontSize:14,fontFamily:'Inter,system-ui,sans-serif'});
  document.querySelectorAll('.mermaid[data-mermaid]').forEach(function(el){
    var code=el.getAttribute('data-mermaid');
    el.removeAttribute('data-processed');el.innerHTML=code;
    mermaid.run({nodes:[el]}).catch(function(e){el.innerHTML='<pre style="color:var(--danger);font-size:12px">Ошибка: '+e.message+'</pre>'});
  });
}
function initPanZoom(){
  document.querySelectorAll('.mermaid-viewport').forEach(function(vp){
    var inner=vp.querySelector('.mermaid-inner'),s=1,px=0,py=0,drag=false,sx,sy,spx,spy;
    function apply(){inner.style.transform='translate('+px+'px,'+py+'px) scale('+s+')';}
    vp.addEventListener('wheel',function(e){e.preventDefault();var r=vp.getBoundingClientRect(),mx=e.clientX-r.left,my=e.clientY-r.top,os=s;s=Math.max(.2,Math.min(5,s*(e.deltaY>0?.9:1.1)));px=mx-(mx-px)*(s/os);py=my-(my-py)*(s/os);apply();},{passive:false});
    vp.addEventListener('mousedown',function(e){drag=true;sx=e.clientX;sy=e.clientY;spx=px;spy=py;});
    window.addEventListener('mousemove',function(e){if(!drag)return;px=spx+(e.clientX-sx);py=spy+(e.clientY-sy);apply();});
    window.addEventListener('mouseup',function(){drag=false;});
    var w=vp.closest('.mermaid-wrapper');
    var zi=w.querySelector('.mz-in'),zo=w.querySelector('.mz-out'),zr=w.querySelector('.mz-reset');
    if(zi)zi.addEventListener('click',function(){s=Math.min(5,s*1.3);apply();});
    if(zo)zo.addEventListener('click',function(){s=Math.max(.2,s*.7);apply();});
    if(zr)zr.addEventListener('click',function(){s=1;px=0;py=0;apply();});
  });
}
// Scroll-spy
(function(){
  var secs=Array.from(document.querySelectorAll('.section'));
  var navs=Array.from(document.querySelectorAll('.nav-item'));
  var tick=false;
  function upd(){
    var y=window.scrollY+120,ai=0;
    for(var i=secs.length-1;i>=0;i--){if(secs[i].offsetTop<=y){ai=i;break;}}
    navs.forEach(function(n,i){
      if(i===ai){n.classList.add('active');var sb=document.getElementById('sidebar'),nr=n.getBoundingClientRect(),sr=sb.getBoundingClientRect();if(nr.top<sr.top+80||nr.bottom>sr.bottom-20)n.scrollIntoView({block:'nearest',behavior:'smooth'});}
      else n.classList.remove('active');
    });tick=false;
  }
  window.addEventListener('scroll',function(){if(!tick){requestAnimationFrame(upd);tick=true;}});
  upd();
})();
document.querySelectorAll('.nav-item').forEach(function(l){l.addEventListener('click',function(e){e.preventDefault();var t=document.querySelector(l.getAttribute('href'));if(t)t.scrollIntoView({behavior:'smooth'});document.querySelector('.sidebar').classList.remove('open');});});
document.addEventListener('DOMContentLoaded',function(){initMermaid('dark');setTimeout(initPanZoom,1500);});
</script>
</body>
</html>`
