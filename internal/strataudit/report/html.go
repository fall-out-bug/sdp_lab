package report

import (
	"fmt"
	"html"
	"net/url"
	"strings"
)

type htmlContext struct {
	documentByID map[string]DocumentReport
	sectionByID  map[string]SectionReport
	entityByID   map[string]EntityReport
}

func buildHTML(rpt *AuditReport) string {
	var b strings.Builder
	projectName := firstNonEmpty(rpt.AuditScope.ProjectName, "StratAudit")
	projectDescription := firstNonEmpty(rpt.AuditScope.ProjectDescription, "Evidence-backed strategy traceability audit")
	htmlLang := firstNonEmpty(rpt.AuditScope.OutputLang, "ru")
	ctx := newHTMLContext(rpt)

	b.WriteString(`<!DOCTYPE html>
<html lang="`)
	b.WriteString(html.EscapeString(htmlLang))
	b.WriteString(`">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>`)
	b.WriteString(html.EscapeString(projectName))
	b.WriteString(` — StratAudit Report</title>
<style>
:root {
  --paper: #f3eee3;
  --paper-deep: #e7dcc8;
  --panel: rgba(255, 251, 245, 0.94);
  --panel-strong: #fffaf2;
  --ink: #201a14;
  --muted: #6e6459;
  --line: #d4c4b0;
  --accent: #a0362c;
  --accent-soft: #ead4c5;
  --accent-cool: #295a72;
  --critical: #9f261f;
  --warning: #a76b00;
  --ok: #215d38;
  --shadow: 0 18px 45px rgba(62, 43, 31, 0.08);
  --mono: "SFMono-Regular", "SF Mono", "Menlo", "Consolas", monospace;
  --serif: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", "Georgia", serif;
  --sans: "Avenir Next", "Segoe UI", "Helvetica Neue", sans-serif;
}
* { box-sizing: border-box; }
html { scroll-behavior: smooth; }
body {
  margin: 0;
  color: var(--ink);
  background:
    radial-gradient(circle at top left, rgba(160, 54, 44, 0.09), transparent 28%),
    radial-gradient(circle at top right, rgba(41, 90, 114, 0.09), transparent 24%),
    linear-gradient(180deg, #fbf7f0, var(--paper) 32%, #efe5d6 100%);
  font-family: var(--sans);
  line-height: 1.55;
}
a { color: var(--accent); text-decoration: none; }
a:hover { text-decoration: underline; }
.page {
  width: min(1440px, calc(100vw - 32px));
  margin: 0 auto;
  padding: 28px 0 48px;
}
.hero {
  position: relative;
  overflow: hidden;
  margin-bottom: 24px;
  padding: 28px 30px 26px;
  border: 1px solid rgba(106, 83, 55, 0.16);
  border-radius: 24px;
  background: linear-gradient(160deg, rgba(255,250,242,0.97), rgba(244,234,220,0.93));
  box-shadow: var(--shadow);
}
.hero::after {
  content: "";
  position: absolute;
  inset: auto -60px -90px auto;
  width: 240px;
  height: 240px;
  border-radius: 999px;
  background: radial-gradient(circle, rgba(160,54,44,0.15), transparent 70%);
}
.eyebrow {
  margin-bottom: 10px;
  color: var(--accent);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}
.hero-top {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
}
.hero-copy {
  max-width: 860px;
}
.hero h1 {
  margin: 0;
  font-family: var(--serif);
  font-size: clamp(34px, 5vw, 56px);
  line-height: 0.95;
  letter-spacing: -0.03em;
}
.hero p {
  margin: 12px 0 0;
  max-width: 860px;
  color: var(--muted);
  font-size: 16px;
}
.status-chip {
  padding: 8px 12px;
  border-radius: 999px;
  border: 1px solid currentColor;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  background: rgba(255,255,255,0.65);
}
.status-critical { color: var(--critical); }
.status-warning { color: var(--warning); }
.status-ok { color: var(--ok); }
.hero-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
  margin-top: 20px;
}
.metric-card {
  min-height: 108px;
  padding: 14px 16px;
  border-radius: 18px;
  border: 1px solid rgba(106, 83, 55, 0.12);
  background: rgba(255,255,255,0.7);
}
.metric-label {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
.metric-value {
  margin-top: 8px;
  font-family: var(--serif);
  font-size: 34px;
  line-height: 1;
}
.metric-note {
  margin-top: 8px;
  color: var(--muted);
  font-size: 13px;
}
.layout {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 20px;
}
.rail {
  position: sticky;
  top: 18px;
  align-self: start;
  padding: 18px;
  border-radius: 20px;
  border: 1px solid rgba(106, 83, 55, 0.12);
  background: rgba(255, 250, 242, 0.88);
  box-shadow: var(--shadow);
}
.rail-title {
  margin: 0 0 12px;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--muted);
}
.rail a {
  display: block;
  padding: 8px 0;
  font-weight: 600;
}
.rail-meta {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--line);
  color: var(--muted);
  font-size: 13px;
}
.main {
  display: grid;
  gap: 18px;
}
.panel {
  padding: 22px 24px;
  border-radius: 22px;
  border: 1px solid rgba(106, 83, 55, 0.12);
  background: var(--panel);
  box-shadow: var(--shadow);
}
.section-kicker {
  margin-bottom: 10px;
  color: var(--accent);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}
.section-title {
  margin: 0;
  font-family: var(--serif);
  font-size: clamp(26px, 3.5vw, 38px);
  line-height: 1;
}
.section-description {
  margin: 12px 0 0;
  color: var(--muted);
}
.panel-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: 18px;
}
.stack {
  display: grid;
  gap: 12px;
  margin-top: 18px;
}
.subpanel {
  padding: 16px 18px;
  border-radius: 18px;
  border: 1px solid rgba(106, 83, 55, 0.12);
  background: var(--panel-strong);
}
.subpanel h3 {
  margin: 0 0 10px;
  font-size: 17px;
}
.subpanel p {
  margin: 0;
  color: var(--muted);
}
.chips, .link-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.chip, .link-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 6px 10px;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: rgba(255,255,255,0.72);
  color: var(--ink);
  font-size: 13px;
}
.chip-critical { border-color: rgba(159,38,31,0.28); color: var(--critical); background: rgba(159,38,31,0.08); }
.chip-warning { border-color: rgba(167,107,0,0.28); color: var(--warning); background: rgba(167,107,0,0.08); }
.chip-ok { border-color: rgba(33,93,56,0.28); color: var(--ok); background: rgba(33,93,56,0.08); }
.pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
.pill-critical { color: var(--critical); background: rgba(159,38,31,0.09); }
.pill-warning { color: var(--warning); background: rgba(167,107,0,0.1); }
.pill-ok { color: var(--ok); background: rgba(33,93,56,0.1); }
.finding-grid,
.trace-grid,
.evidence-grid,
.ledger {
  display: grid;
  gap: 14px;
  margin-top: 18px;
}
.finding-card,
.trace-card,
.evidence-card,
.ledger-card {
  padding: 18px;
  border-radius: 20px;
  border: 1px solid rgba(106, 83, 55, 0.12);
  background: rgba(255,255,255,0.78);
}
.finding-head,
.trace-head,
.evidence-head,
.ledger-head {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
}
.card-title {
  margin: 0;
  font-size: 19px;
  line-height: 1.2;
}
.card-meta {
  margin-top: 8px;
  color: var(--muted);
  font-size: 13px;
}
.card-body {
  margin-top: 14px;
  color: var(--ink);
}
.card-body p {
  margin: 0;
}
.split {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 14px;
}
.quote-box {
  padding: 14px;
  border-radius: 16px;
  background: rgba(239,229,214,0.7);
  border: 1px solid rgba(106, 83, 55, 0.12);
}
.quote-box h4 {
  margin: 0 0 8px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.quote-box pre {
  margin: 0;
  white-space: pre-wrap;
  font-family: var(--sans);
  font-size: 14px;
}
.table-wrap {
  margin-top: 14px;
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}
th, td {
  padding: 12px 10px;
  border-bottom: 1px solid var(--line);
  text-align: left;
  vertical-align: top;
}
th {
  color: var(--muted);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.bar {
  position: relative;
  width: 140px;
  height: 9px;
  border-radius: 999px;
  background: rgba(32,26,20,0.08);
  overflow: hidden;
}
.bar > span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--accent), var(--accent-cool));
}
.doc-link {
  font-family: var(--mono);
  font-size: 13px;
}
.muted {
  color: var(--muted);
}
.artifacts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
  margin-top: 16px;
}
.artifact-card {
  padding: 14px;
  border-radius: 18px;
  background: rgba(255,255,255,0.8);
  border: 1px solid rgba(106, 83, 55, 0.12);
}
.artifact-kind {
  color: var(--muted);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.artifact-path {
  margin-top: 8px;
  font-family: var(--mono);
  font-size: 12px;
  overflow-wrap: anywhere;
}
.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 18px;
}
.filter-btn {
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: rgba(255,255,255,0.74);
  color: var(--ink);
  font-size: 13px;
  font-weight: 700;
}
.filter-btn.active { border-color: var(--accent); color: var(--accent); background: rgba(160,54,44,0.08); }
.empty-note {
  padding: 14px;
  border-radius: 16px;
  border: 1px dashed var(--line);
  color: var(--muted);
}
@media (max-width: 1100px) {
  .layout { grid-template-columns: 1fr; }
  .rail { position: static; }
  .hero-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .panel-grid, .split { grid-template-columns: 1fr; }
}
@media (max-width: 720px) {
  .page { width: min(100vw - 20px, 100%); padding-top: 12px; }
  .hero { padding: 20px; border-radius: 20px; }
  .panel { padding: 18px; }
  .hero-grid { grid-template-columns: 1fr; }
  .metric-value { font-size: 28px; }
}
</style>
</head>
<body>
<div class="page">
  <header class="hero">
    <div class="eyebrow">Evidence-backed Strategy Audit</div>
    <div class="hero-top">
      <div class="hero-copy">
        <h1>`)
	b.WriteString(html.EscapeString(projectName))
	b.WriteString(`</h1>
        <p>`)
	b.WriteString(html.EscapeString(projectDescription))
	b.WriteString(`</p>
      </div>
      <div class="status-chip status-`)
	b.WriteString(html.EscapeString(statusClass(rpt.TrustSummary.OverallStatus)))
	b.WriteString(`">`)
	b.WriteString(html.EscapeString(statusLabel(rpt.TrustSummary.OverallStatus)))
	b.WriteString(`</div>
    </div>
    <div class="hero-grid">`)
	writeMetricCard(&b, "Verified entities", fmt.Sprintf("%d", rpt.TrustSummary.Entities.Verified), fmt.Sprintf("%d rejected / %d suspect", rpt.TrustSummary.Entities.Rejected, rpt.TrustSummary.Entities.Suspect))
	writeMetricCard(&b, "Verified traces", fmt.Sprintf("%d", rpt.TrustSummary.Traces.Verified), fmt.Sprintf("%d candidates", rpt.TrustSummary.Traces.Candidates))
	writeMetricCard(&b, "Critical findings", fmt.Sprintf("%d", rpt.TrustSummary.Findings.Critical), fmt.Sprintf("%d total findings", rpt.TrustSummary.Findings.Total))
	writeMetricCard(&b, "Critical corpus docs", fmt.Sprintf("%d", rpt.CorpusQuality.CriticalDocuments), fmt.Sprintf("%d total corpus issues", rpt.CorpusQuality.TotalIssues))
	writeMetricCard(&b, "Average level coverage", fmt.Sprintf("%.0f%%", rpt.Coverage.AverageLevelPct), fmt.Sprintf("%d documents / %d sections", len(rpt.Documents), len(rpt.Sections)))
	b.WriteString(`</div>
  </header>

  <div class="layout">
    <aside class="rail">
      <div class="rail-title">Report Layers</div>
      <a href="#executive-overview">Executive Overview</a>
      <a href="#analyst-explorer">Analyst Explorer</a>
      <a href="#evidence-pack">Evidence Pack</a>
      <div class="rail-meta">
        <div>Generated: `)
	b.WriteString(html.EscapeString(rpt.AuditScope.GeneratedAt))
	b.WriteString(`</div>
        <div>Schema: v`)
	b.WriteString(html.EscapeString(rpt.SchemaVersion))
	b.WriteString(`</div>
        <div>Output: `)
	b.WriteString(html.EscapeString(firstNonEmpty(rpt.AuditScope.OutputDir, ".strataudit")))
	b.WriteString(`</div>
      </div>
    </aside>

    <main class="main">`)
	renderExecutiveOverview(&b, rpt, ctx)
	renderAnalystExplorer(&b, rpt, ctx)
	renderEvidencePack(&b, rpt, ctx)
	b.WriteString(`
    </main>
  </div>
</div>
<script>
document.querySelectorAll('[data-filter-btn]').forEach(function(btn) {
  btn.addEventListener('click', function(event) {
    var target = event.currentTarget;
    var filter = target.getAttribute('data-filter-btn');
    document.querySelectorAll('[data-filter-btn]').forEach(function(other) {
      other.classList.remove('active');
    });
    target.classList.add('active');
    document.querySelectorAll('[data-finding-type]').forEach(function(card) {
      var cardType = card.getAttribute('data-finding-type');
      card.style.display = (filter === 'all' || cardType === filter) ? '' : 'none';
    });
  });
});
</script>
</body>
</html>`)

	return b.String()
}

func renderExecutiveOverview(b *strings.Builder, rpt *AuditReport, ctx *htmlContext) {
	b.WriteString(`
      <section class="panel" id="executive-overview">
        <div class="section-kicker">Layer 1</div>
        <h2 class="section-title">Executive Overview</h2>
        <p class="section-description">Сверху только то, что помогает быстро понять, можно ли доверять аудиту, где блокеры корпуса и какие документы сильнее всего рвут стратегическую трассировку.</p>
`)
	if len(rpt.TrustSummary.Disclaimers) > 0 {
		b.WriteString(`<div class="stack">`)
		for _, disclaimer := range rpt.TrustSummary.Disclaimers {
			fmt.Fprintf(b, `<div class="subpanel"><h3>Trust disclaimer</h3><p>%s</p></div>`, html.EscapeString(disclaimer))
		}
		b.WriteString(`</div>`)
	}

	b.WriteString(`<div class="panel-grid">`)
	b.WriteString(`<div class="subpanel"><h3>Corpus Quality Blockers</h3>`)
	if len(rpt.CorpusQuality.Documents) == 0 {
		b.WriteString(`<div class="empty-note">Критических corpus-quality blockers не найдено.</div>`)
	} else {
		b.WriteString(`<div class="ledger">`)
		limit := minInt(3, len(rpt.CorpusQuality.Documents))
		for _, doc := range rpt.CorpusQuality.Documents[:limit] {
			fmt.Fprintf(b, `<div class="ledger-card"><div class="ledger-head"><div><div class="card-title">%s</div><div class="card-meta">%s · %d issues</div></div><span class="pill pill-%s">%s</span></div><div class="card-body"><div class="chips">%s</div><div class="link-list" style="margin-top:12px">%s %s</div></div></div>`,
				html.EscapeString(doc.DocumentPath),
				html.EscapeString(firstNonEmpty(doc.LevelName, doc.LevelID)),
				doc.IssueCount,
				html.EscapeString(statusClass(doc.Severity)),
				html.EscapeString(strings.ToUpper(doc.Severity)),
				renderFlagChips(doc.Flags),
				internalLink(fmt.Sprintf("#doc-%s", doc.DocumentID), "Open document dossier"),
				localDocumentLink(doc.DocumentPath, "Open file"))
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="subpanel"><h3>Lowest Coverage</h3>`)
	if len(rpt.Coverage.LowestDocuments) == 0 {
		b.WriteString(`<div class="empty-note">Нет document-level coverage данных.</div>`)
	} else {
		b.WriteString(`<div class="ledger">`)
		limit := minInt(3, len(rpt.Coverage.LowestDocuments))
		for _, item := range rpt.Coverage.LowestDocuments[:limit] {
			fmt.Fprintf(b, `<div class="ledger-card"><div class="ledger-head"><div><div class="card-title">%s</div><div class="card-meta">%s · %d/%d traced</div></div><span class="pill pill-%s">%.0f%%</span></div><div class="link-list" style="margin-top:12px">%s %s</div></div>`,
				html.EscapeString(firstNonEmpty(item.ScopeLabel, item.DocumentPath, item.ScopeID)),
				html.EscapeString(firstNonEmpty(item.LevelName, item.LevelID)),
				item.Traced,
				item.Total,
				html.EscapeString(coverageClass(item.Pct)),
				item.Pct,
				internalLink(fmt.Sprintf("#doc-%s", item.DocumentID), "Open document dossier"),
				internalLink("#coverage-explorer", "Coverage explorer"))
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)
	b.WriteString(`</section>`)
}

func renderAnalystExplorer(b *strings.Builder, rpt *AuditReport, ctx *htmlContext) {
	b.WriteString(`
      <section class="panel" id="analyst-explorer">
        <div class="section-kicker">Layer 2</div>
        <h2 class="section-title">Analyst Explorer</h2>
        <p class="section-description">Отсюда аналитик должен пройти путь от findings и traces до документа, section и quote evidence без обращения к SQLite или внешнему дашборду.</p>

        <div class="filter-row">
          <button class="filter-btn active" data-filter-btn="all">All findings</button>
          <button class="filter-btn" data-filter-btn="strategic_gap_cluster">Strategic gaps</button>
          <button class="filter-btn" data-filter-btn="orphan_cluster">Orphans</button>
          <button class="filter-btn" data-filter-btn="corpus_quality_cluster">Corpus quality</button>
          <button class="filter-btn" data-filter-btn="trace_ambiguity_cluster">Trace ambiguity</button>
        </div>
        <div class="finding-grid">`)
	if len(rpt.FindingsGrouped) == 0 {
		b.WriteString(`<div class="empty-note">Grouped findings отсутствуют.</div>`)
	} else {
		for _, finding := range rpt.FindingsGrouped {
			fmt.Fprintf(b, `<article class="finding-card" id="finding-%s" data-finding-type="%s"><div class="finding-head"><div><h3 class="card-title">%s</h3><div class="card-meta">%s · confidence %.0f%% · affected %d</div></div><span class="pill pill-%s">%s</span></div><div class="card-body"><p>%s</p>`,
				html.EscapeString(finding.ID),
				html.EscapeString(finding.Type),
				html.EscapeString(finding.Title),
				html.EscapeString(finding.Type),
				finding.Confidence*100,
				finding.AffectedCount,
				html.EscapeString(statusClass(finding.Severity)),
				html.EscapeString(strings.ToUpper(finding.Severity)),
				html.EscapeString(finding.Description))
			if strings.TrimSpace(finding.Recommendation) != "" {
				fmt.Fprintf(b, `<p style="margin-top:10px" class="muted">Recommendation: %s</p>`, html.EscapeString(finding.Recommendation))
			}
			b.WriteString(`<div class="link-list" style="margin-top:14px">`)
			for _, documentID := range finding.DocumentIDs {
				label := documentID
				if doc, ok := ctx.documentByID[documentID]; ok {
					label = firstNonEmpty(doc.Name, doc.Path, documentID)
				}
				b.WriteString(internalLink(fmt.Sprintf("#doc-%s", documentID), "Doc: "+label))
			}
			for _, sectionID := range finding.SectionIDs {
				label := sectionID
				if section, ok := ctx.sectionByID[sectionID]; ok {
					label = section.Label
				}
				b.WriteString(internalLink(fmt.Sprintf("#section-%s", sectionID), "Section: "+label))
			}
			for _, entityID := range finding.EntityIDs {
				label := entityID
				if entity, ok := ctx.entityByID[entityID]; ok {
					label = firstNonEmpty(entity.TitleOriginal, entity.Title, entityID)
				}
				b.WriteString(internalLink(fmt.Sprintf("#evidence-entity-%s", entityID), "Evidence: "+label))
			}
			b.WriteString(`</div></div></article>`)
		}
	}
	b.WriteString(`</div>

        <div class="stack" id="trace-explorer">
          <div class="subpanel">
            <h3>Trace Explorer</h3>
            <p>Verified traces показываются отдельно от similarity candidates. Каждая карточка держит source/target evidence и путь к документу.</p>
          </div>
          <div class="trace-grid">`)
	if len(rpt.VerifiedTraces) == 0 {
		b.WriteString(`<div class="empty-note">Verified traces отсутствуют.</div>`)
	} else {
		for _, trace := range rpt.VerifiedTraces {
			sourceTitle := trace.SourceEntityID
			if entity, ok := ctx.entityByID[trace.SourceEntityID]; ok {
				sourceTitle = firstNonEmpty(entity.TitleOriginal, entity.Title, trace.SourceEntityID)
			}
			targetTitle := trace.TargetEntityID
			if entity, ok := ctx.entityByID[trace.TargetEntityID]; ok {
				targetTitle = firstNonEmpty(entity.TitleOriginal, entity.Title, trace.TargetEntityID)
			}
			fmt.Fprintf(b, `<article class="trace-card" id="trace-%s"><div class="trace-head"><div><h3 class="card-title">%s → %s</h3><div class="card-meta">%s · %s · confidence %.0f%% · similarity %.2f</div></div><span class="pill pill-%s">%s</span></div><div class="card-body"><p>%s</p><div class="split">%s%s</div></div></article>`,
				html.EscapeString(trace.ID),
				html.EscapeString(sourceTitle),
				html.EscapeString(targetTitle),
				html.EscapeString(trace.Relation),
				html.EscapeString(trace.VerificationMode),
				trace.Confidence*100,
				trace.SimilarityScore,
				html.EscapeString(statusClass(trace.TrustGrade)),
				html.EscapeString(strings.ToUpper(firstNonEmpty(trace.TrustGrade, "verified"))),
				html.EscapeString(trace.Justification),
				renderTraceEvidenceBox("Source evidence", trace.TraceEvidence.Source, ctx),
				renderTraceEvidenceBox("Target evidence", trace.TraceEvidence.Target, ctx))
		}
	}
	if len(rpt.TraceCandidates) > 0 {
		fmt.Fprintf(b, `<div class="subpanel"><h3>Diagnostic candidates</h3><p>%d similarity candidates остались диагностикой и не выдаются за proof.</p></div>`, len(rpt.TraceCandidates))
	}
	b.WriteString(`</div>
        </div>

        <div class="stack" id="coverage-explorer">
          <div class="subpanel">
            <h3>Coverage Explorer</h3>
            <p>Coverage раскрывается на трёх уровнях: level, document, section. Это нужно, чтобы не принимать один усреднённый процент за доказательство качества стратегии.</p>
          </div>
`)
	renderCoverageTable(b, "Coverage by Level", rpt.Coverage.ByLevel, ctx)
	renderCoverageTable(b, "Coverage by Document", rpt.Coverage.ByDocument, ctx)
	renderCoverageTable(b, "Coverage by Section", rpt.Coverage.BySection, ctx)
	b.WriteString(`</div>
      </section>`)
}

func renderEvidencePack(b *strings.Builder, rpt *AuditReport, ctx *htmlContext) {
	b.WriteString(`
      <section class="panel" id="evidence-pack">
        <div class="section-kicker">Layer 3</div>
        <h2 class="section-title">Evidence Pack</h2>
        <p class="section-description">Полный offline-dossier: документы, section previews, entity evidence и generated artifacts. Это не dashboard; это пакет для верификации.</p>
`)
	b.WriteString(`<div class="stack">`)

	b.WriteString(`<div class="subpanel"><h3>Artifacts</h3><div class="artifacts">`)
	for _, artifact := range rpt.EvidencePack.Artifacts {
		fmt.Fprintf(b, `<div class="artifact-card"><div class="artifact-kind">%s</div><div class="artifact-path">%s</div><div class="link-list" style="margin-top:10px">%s</div></div>`,
			html.EscapeString(artifact.Kind),
			html.EscapeString(artifact.Path),
			localDocumentLink(artifact.Path, "Open artifact"))
	}
	b.WriteString(`</div></div>`)

	b.WriteString(`<div class="subpanel"><h3>Document Dossier</h3><div class="ledger">`)
	if len(rpt.Documents) == 0 {
		b.WriteString(`<div class="empty-note">Документы не загружены в evidence pack.</div>`)
	} else {
		for _, doc := range rpt.Documents {
			fmt.Fprintf(b, `<article class="ledger-card" id="doc-%s"><div class="ledger-head"><div><h4 class="card-title">%s</h4><div class="card-meta">%s · %d entities · %d sections</div></div><span class="pill pill-%s">%s</span></div><div class="card-body"><div class="link-list">%s %s</div></div></article>`,
				html.EscapeString(doc.ID),
				html.EscapeString(doc.Path),
				html.EscapeString(firstNonEmpty(doc.LevelName, doc.LevelID)),
				doc.EntityCount,
				doc.SectionCount,
				html.EscapeString(documentHealthClass(doc, rpt)),
				html.EscapeString(strings.ToUpper(documentHealthLabel(doc, rpt))),
				localDocumentLink(doc.Path, "Open file"),
				internalLink("#coverage-explorer", "Coverage"))
		}
	}
	b.WriteString(`</div></div>`)

	b.WriteString(`<div class="subpanel"><h3>Section Evidence</h3><div class="evidence-grid">`)
	if len(rpt.Sections) == 0 {
		b.WriteString(`<div class="empty-note">Sections отсутствуют.</div>`)
	} else {
		for _, section := range rpt.Sections {
			fmt.Fprintf(b, `<article class="evidence-card" id="section-%s"><div class="evidence-head"><div><h4 class="card-title">%s</h4><div class="card-meta">%s · chars %d-%d · %d entities</div></div><div class="chips">%s</div></div><div class="card-body"><div class="quote-box"><h4>Section preview</h4><pre>%s</pre></div><div class="link-list" style="margin-top:12px">%s %s</div></div></article>`,
				html.EscapeString(section.ID),
				html.EscapeString(section.Label),
				html.EscapeString(firstNonEmpty(section.LevelName, section.LevelID)),
				section.CharStart,
				section.CharEnd,
				section.EntityCount,
				renderFlagChips(section.QualityFlags),
				html.EscapeString(section.Preview),
				internalLink(fmt.Sprintf("#doc-%s", section.DocumentID), "Document dossier"),
				localDocumentLink(section.DocumentPath, "Open file"))
		}
	}
	b.WriteString(`</div></div>`)

	b.WriteString(`<div class="subpanel"><h3>Entity Evidence Cards</h3><div class="evidence-grid">`)
	if len(rpt.Entities) == 0 {
		b.WriteString(`<div class="empty-note">Admitted entities отсутствуют.</div>`)
	} else {
		for _, entity := range rpt.Entities {
			fmt.Fprintf(b, `<article class="evidence-card" id="evidence-entity-%s"><div class="evidence-head"><div><h4 class="card-title">%s</h4><div class="card-meta">%s · %s · %s</div></div><span class="pill pill-%s">%s</span></div><div class="card-body"><p class="muted">%s</p><div class="quote-box" style="margin-top:12px"><h4>Source quote</h4><pre>%s</pre></div><div class="link-list" style="margin-top:12px">%s %s %s</div></div></article>`,
				html.EscapeString(entity.ID),
				html.EscapeString(firstNonEmpty(entity.TitleOriginal, entity.Title)),
				html.EscapeString(entity.Type),
				html.EscapeString(firstNonEmpty(entity.LevelName, entity.LevelID)),
				html.EscapeString(sectionLabelForEntity(entity, ctx)),
				html.EscapeString(statusClass(entity.TrustGrade)),
				html.EscapeString(strings.ToUpper(firstNonEmpty(entity.TrustGrade, "verified"))),
				html.EscapeString(firstNonEmpty(entity.DescriptionOriginal, entity.Description)),
				html.EscapeString(entity.SourceQuote),
				internalLink(fmt.Sprintf("#section-%s", entity.SectionID), "Section"),
				internalLink(fmt.Sprintf("#doc-%s", entity.DocumentID), "Document"),
				localDocumentLink(entity.DocumentPath, "Open file"))
		}
	}
	b.WriteString(`</div></div>`)

	b.WriteString(`</div></section>`)
}

func renderCoverageTable(b *strings.Builder, title string, entries []CoverageScopeReport, ctx *htmlContext) {
	b.WriteString(`<div class="subpanel"><h3>`)
	b.WriteString(html.EscapeString(title))
	b.WriteString(`</h3>`)
	if len(entries) == 0 {
		b.WriteString(`<div class="empty-note">Coverage data отсутствуют.</div></div>`)
		return
	}
	b.WriteString(`<div class="table-wrap"><table><thead><tr><th>Scope</th><th>Coverage</th><th>Traced</th><th>Links</th></tr></thead><tbody>`)
	for _, entry := range entries {
		fmt.Fprintf(b, `<tr><td><strong>%s</strong><div class="muted">%s</div></td><td><div class="bar"><span style="width: %.2f%%"></span></div><div class="muted" style="margin-top:6px">%.0f%%</div></td><td>%d / %d</td><td><div class="link-list">%s</div></td></tr>`,
			html.EscapeString(firstNonEmpty(entry.ScopeLabel, entry.DocumentPath, entry.ScopeID)),
			html.EscapeString(firstNonEmpty(entry.LevelName, entry.LevelID)),
			entry.Pct,
			entry.Pct,
			entry.Traced,
			entry.Total,
			renderCoverageLinks(entry, ctx))
	}
	b.WriteString(`</tbody></table></div></div>`)
}

func renderCoverageLinks(entry CoverageScopeReport, ctx *htmlContext) string {
	var parts []string
	if entry.DocumentID != "" {
		parts = append(parts, internalLink(fmt.Sprintf("#doc-%s", entry.DocumentID), "Document"))
		if doc, ok := ctx.documentByID[entry.DocumentID]; ok {
			parts = append(parts, localDocumentLink(doc.Path, "Open file"))
		}
	}
	if entry.SectionID != "" {
		parts = append(parts, internalLink(fmt.Sprintf("#section-%s", entry.SectionID), "Section"))
	}
	if len(parts) == 0 {
		return `<span class="muted">No direct drill-down</span>`
	}
	return strings.Join(parts, " ")
}

func renderTraceEvidenceBox(title string, ref EvidenceRefReport, ctx *htmlContext) string {
	return fmt.Sprintf(`<div class="quote-box"><h4>%s</h4><div class="card-meta">%s · %s</div><pre>%s</pre><div class="link-list" style="margin-top:12px">%s %s</div></div>`,
		html.EscapeString(title),
		html.EscapeString(ref.DocumentPath),
		html.EscapeString(sectionLabelForEvidence(ref, ctx)),
		html.EscapeString(ref.Quote),
		internalLink(fmt.Sprintf("#section-%s", ref.SectionID), "Section"),
		localDocumentLink(ref.DocumentPath, "Open file"))
}

func renderFlagChips(flags []string) string {
	if len(flags) == 0 {
		return `<span class="chip">no flags</span>`
	}
	var b strings.Builder
	for _, flag := range flags {
		fmt.Fprintf(&b, `<span class="chip %s">%s</span>`, severityChipClass(flagSeverity(flag)), html.EscapeString(flag))
	}
	return b.String()
}

func newHTMLContext(rpt *AuditReport) *htmlContext {
	ctx := &htmlContext{
		documentByID: make(map[string]DocumentReport, len(rpt.Documents)),
		sectionByID:  make(map[string]SectionReport, len(rpt.Sections)),
		entityByID:   make(map[string]EntityReport, len(rpt.Entities)),
	}
	for _, doc := range rpt.Documents {
		ctx.documentByID[doc.ID] = doc
	}
	for _, section := range rpt.Sections {
		ctx.sectionByID[section.ID] = section
	}
	for _, entity := range rpt.Entities {
		ctx.entityByID[entity.ID] = entity
	}
	return ctx
}

func writeMetricCard(b *strings.Builder, label, value, note string) {
	fmt.Fprintf(b, `<div class="metric-card"><div class="metric-label">%s</div><div class="metric-value">%s</div><div class="metric-note">%s</div></div>`,
		html.EscapeString(label), html.EscapeString(value), html.EscapeString(note))
}

func internalLink(target, label string) string {
	return fmt.Sprintf(`<a class="link-chip" href="%s">%s</a>`, html.EscapeString(target), html.EscapeString(label))
}

func localDocumentLink(path, label string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return fmt.Sprintf(`<a class="link-chip doc-link" href="%s">%s</a>`, html.EscapeString(fileURL(path)), html.EscapeString(label))
}

func fileURL(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func statusClass(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "critical", "rejected":
		return "critical"
	case "warning", "warn", "suspect":
		return "warning"
	default:
		return "ok"
	}
}

func statusLabel(status string) string {
	switch statusClass(status) {
	case "critical":
		return "critical"
	case "warning":
		return "warning"
	default:
		return "ok"
	}
}

func coverageClass(pct float64) string {
	if pct < 50 {
		return "critical"
	}
	if pct < 70 {
		return "warning"
	}
	return "ok"
}

func documentHealthClass(doc DocumentReport, rpt *AuditReport) string {
	for _, issue := range rpt.CorpusQuality.Documents {
		if issue.DocumentID == doc.ID {
			return statusClass(issue.Severity)
		}
	}
	return coverageClass(documentCoveragePct(doc.ID, rpt.Coverage.ByDocument))
}

func documentHealthLabel(doc DocumentReport, rpt *AuditReport) string {
	for _, issue := range rpt.CorpusQuality.Documents {
		if issue.DocumentID == doc.ID {
			return issue.Severity
		}
	}
	return coverageClass(documentCoveragePct(doc.ID, rpt.Coverage.ByDocument))
}

func documentCoveragePct(documentID string, entries []CoverageScopeReport) float64 {
	for _, entry := range entries {
		if entry.DocumentID == documentID {
			return entry.Pct
		}
	}
	return 100
}

func sectionLabelForEvidence(ref EvidenceRefReport, ctx *htmlContext) string {
	if section, ok := ctx.sectionByID[ref.SectionID]; ok {
		return section.Label
	}
	if ref.SectionHeading != "" {
		return ref.SectionHeading
	}
	return ref.SectionID
}

func sectionLabelForEntity(entity EntityReport, ctx *htmlContext) string {
	if section, ok := ctx.sectionByID[entity.SectionID]; ok {
		return section.Label
	}
	if entity.SectionHeading != "" {
		return entity.SectionHeading
	}
	return entity.SectionID
}

func flagSeverity(flag string) string {
	switch flag {
	case "prompt_leak", "quote_not_found", "boilerplate_repetition", "language_mismatch":
		return "critical"
	case "section_parse_fallback", "html_export_noise", "mime_header_noise":
		return "warning"
	default:
		return "ok"
	}
}

func severityChipClass(severity string) string {
	switch severity {
	case "critical":
		return "chip-critical"
	case "warning":
		return "chip-warning"
	default:
		return "chip-ok"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
