package report

import (
	"fmt"
	"html"
	"strings"
)

func buildHTML(rpt *AuditReport) string {
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>`)
	b.WriteString(html.EscapeString(rpt.Project))
	b.WriteString(` — StratAudit Report</title>
<style>
:root {
  --bg: #0f0f0f; --surface: #1a1a1a; --surface2: #252525;
  --text: #e8e8e8; --text-muted: #888; --accent: #4760F3;
  --cw-success: #10B981; --cw-warning: #F59E0B; --cw-danger: #EF4444;
  --radius: 8px; --radius-lg: 16px;
  --shadow: 0 4px 24px rgba(71,96,243,0.08);
  --font: system-ui, -apple-system, sans-serif;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: var(--font); background: var(--bg); color: var(--text); line-height: 1.6; padding: 2rem; max-width: 1200px; margin: 0 auto; }
h1 { font-size: 2rem; font-weight: 700; margin-bottom: 0.5rem; }
h2 { font-size: 1.3rem; font-weight: 600; margin: 2rem 0 1rem; color: var(--accent); }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 1rem; margin: 1.5rem 0; }
.stat { background: var(--surface); border-radius: var(--radius); padding: 1.25rem; border: 1px solid #2a2a2a; }
.stat-label { font-size: 0.8rem; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; }
.stat-value { font-size: 1.8rem; font-weight: 700; margin-top: 0.25rem; }
.critical .stat-value { color: var(--cw-danger); }
.warning .stat-value { color: var(--cw-warning); }
.ok .stat-value { color: var(--cw-success); }
.finding { background: var(--surface); border-radius: var(--radius); padding: 1rem 1.25rem; margin-bottom: 0.75rem; border-left: 4px solid var(--accent); }
.finding.severity-critical { border-left-color: var(--cw-danger); }
.finding.severity-warn { border-left-color: var(--cw-warning); }
.finding.severity-info { border-left-color: var(--cw-success); }
.finding-type { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.08em; color: var(--accent); font-weight: 600; }
.finding-title { font-weight: 600; margin: 0.25rem 0; }
.finding-desc { font-size: 0.9rem; color: var(--text-muted); }
.confidence { font-size: 0.75rem; padding: 2px 8px; border-radius: 4px; background: var(--surface2); color: var(--text-muted); display: inline-block; }
.filter-bar { display: flex; gap: 0.5rem; flex-wrap: wrap; margin: 1rem 0; }
.filter-btn { padding: 0.4rem 0.8rem; border-radius: var(--radius); border: 1px solid #333; background: var(--surface); color: var(--text); cursor: pointer; font-size: 0.85rem; }
.filter-btn.active { background: var(--accent); border-color: var(--accent); color: white; }
.subtitle { color: var(--text-muted); font-size: 0.95rem; margin-bottom: 1.5rem; }
</style>
</head>
<body>
<h1>`)

	b.WriteString(html.EscapeString(rpt.Project))
	b.WriteString(`</h1>
<p class="subtitle">Strategy Traceability Audit Report</p>

<div class="stats">
  <div class="stat ok"><div class="stat-label">Entities</div><div class="stat-value">`)
	fmt.Fprintf(&b, "%d", rpt.Summary.TotalEntities)
	b.WriteString(`</div></div>
  <div class="stat"><div class="stat-label">Findings</div><div class="stat-value">`)
	fmt.Fprintf(&b, "%d", rpt.Summary.TotalFindings)
	b.WriteString(`</div></div>
  <div class="stat critical"><div class="stat-label">Critical</div><div class="stat-value">`)
	fmt.Fprintf(&b, "%d", rpt.Summary.CriticalCount)
	b.WriteString(`</div></div>
  <div class="stat warning"><div class="stat-label">Warnings</div><div class="stat-value">`)
	fmt.Fprintf(&b, "%d", rpt.Summary.WarnCount)
	b.WriteString(`</div></div>
  <div class="stat ok"><div class="stat-label">Avg Coverage</div><div class="stat-value">`)
	fmt.Fprintf(&b, "%.0f%%", rpt.Summary.AvgCoverage)
	b.WriteString(`</div></div>
</div>

<h2>Coverage by Level</h2>
<div class="stats">
`)

	for _, c := range rpt.Coverage {
		cls := "ok"
		if c.Pct < 50 {
			cls = "critical"
		} else if c.Pct < 70 {
			cls = "warning"
		}
		fmt.Fprintf(&b, `  <div class="stat %s"><div class="stat-label">%s</div><div class="stat-value">%.0f%%</div><div style="font-size:0.8rem;color:var(--text-muted)">%d/%d traced</div></div>
`, cls, html.EscapeString(c.Level), c.Pct, c.Traced, c.Total)
	}

	b.WriteString(`</div>

<h2>Findings</h2>
<div class="filter-bar">
  <button class="filter-btn active" onclick="filterFindings('all')">All</button>
  <button class="filter-btn" onclick="filterFindings('gap')">Gaps</button>
  <button class="filter-btn" onclick="filterFindings('orphan')">Orphans</button>
  <button class="filter-btn" onclick="filterFindings('alignment')">Alignment</button>
  <button class="filter-btn" onclick="filterFindings('coverage')">Coverage</button>
  <button class="filter-btn" onclick="filterFindings('unknown_rationale')">Unknown</button>
</div>
<div id="findings">
`)

	for _, f := range rpt.Findings {
		tier := "low"
		if f.Confidence >= 0.7 {
			tier = "high"
		} else if f.Confidence >= 0.4 {
			tier = "medium"
		}
		fmt.Fprintf(&b, `<div class="finding severity-%s" data-type="%s">
  <span class="finding-type">%s</span>
  <div class="finding-title">%s</div>
  <div class="finding-desc">%s</div>
  <span class="confidence %s">%.0f%%</span>
</div>
`, html.EscapeString(f.Severity), html.EscapeString(f.Type),
			html.EscapeString(f.Type), html.EscapeString(f.Title),
			html.EscapeString(f.Description), tier, f.Confidence*100)
	}

	b.WriteString(`</div>

<script>
function filterFindings(type) {
  document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
  event.target.classList.add('active');
  document.querySelectorAll('.finding').forEach(f => {
    f.style.display = (type === 'all' || f.dataset.type === type) ? '' : 'none';
  });
}
</script>
</body>
</html>`)

	return b.String()
}
