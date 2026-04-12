# SDP 12-Week OSS Growth Playbook

**Target Protocol:** SDP (Structured Development Protocol) — Evidence layer for AI agent workflows  
**Current State:** 16 stars, 0 forks, 3 open issues, 10 releases in 2 months (v0.4.0→v0.9.8)  
**Created:** 2025-12-28  
**Playbook Author:** Research-backed execution plan based on successful OSS tools (OPA, Kyverno, k9s, in-toto, GUAC, GitHub CLI)  
**Execution Start:** Week 1 of playbook implementation

---

## Executive Summary

### The Problem Space

AI agents write code, but no tool proves they followed a process. SDP provides the missing **evidence layer**: machine-readable proof of intent, execution, verification, and review for every agent-produced change.

### Current State Analysis

| Metric | Current | Target (12 weeks) | Gap |
|--------|---------|-------------------|-----|
| Stars | 16 | 500 | +484 |
| Forks | 0 | 50 | +50 |
| Weekly releases | 3-4 | 2-3 (steady cadence) | Maintain |
| CLI downloads | 0 | 200/month | +200 |
| Issue engagement | 3 open | 20 open, 10 closed | +17/+10 |
| Documentation coverage | ~60% | 90% | +30% |
| CI integration examples | 0 | 5 language examples | +5 |

### Core Hypothesis

**SDP will grow by becoming the de facto evidence standard for AI coding agents.** The playbook follows the proven pattern of standards-based CLI tools (OPA, Kyverno) that achieved 1,000+ stars through:

1. **CLI-first entry point** — `sdp-evidence` as the flagship tool
2. **Standards-based approach** — in-toto attestation format (Phase 3)
3. **CI/CD integration** — evidence gate as the primary use case
4. **Community hooks** — OpenCode, OhMyOpenCode, Beads, Gas Town adapters
5. **Documentation excellence** — comprehensive guides and examples

### Success Definition

After 12 weeks, SDP is a **production-ready evidence layer** for AI coding agents with:
- 500+ GitHub stars (organic growth, not astroturfed)
- 50+ forks (indicating experimentation)
- 200+ monthly CLI downloads
- 5+ language examples (Go, Python, TypeScript, Rust, Java)
- 1+ external adapter (OhMyOpenCode, Beads, Gas Town) published
- 20+ open issues (community engagement)
- Evidence gate CI job available for 5+ CI systems

---

## Research Findings: Successful OSS Developer Tools

### Case Study 1: Open Policy Agent (OPA) — 11,292 stars

**Pattern:** Standards-based policy engine with CLI-first adoption

| What OPA Did | Frequency | Result |
|--------------|-----------|--------|
| Stable CLI releases | Monthly | +500 stars/month in growth phase |
| Policy examples library | 50+ examples | Low barrier to entry |
| CI/CD integrations | 10+ CI platforms | Enterprise adoption |
| Documentation | docs.openpolicyagent.org | SEO dominance |
| Community policies | 200+ Rego policies | Network effects |

**Evidence:** [OPA GitHub repo](https://github.com/open-policy-agent/opa) — 11,292 stars, 1,256 forks, created 2015-12-28. Release cadence: v1.14.0 (Feb 2026), v1.13.0 (Jan 2026) — **monthly releases during growth**.

### Case Study 2: Kyverno — 7,453 stars

**Pattern:** K8s admission controller with policy-as-code

| What Kyverno Did | Frequency | Result |
|------------------|-----------|--------|
| K8s admission controller | v1.17.1 (Feb 2026) | Native K8s ecosystem fit |
| Policy marketplace | 300+ policies | Community contribution |
| CLI validation tool | `kyverno-cli` | Local testing workflow |
| Documentation | kyverno.io/docs | SEO + discoverability |
| Release notes | Detailed changelogs | Trust building |

**Evidence:** [Kyverno GitHub repo](https://github.com/kyverno/kyverno) — 7,453 stars, 1,002 forks, created 2019-02-04. Release cadence: v1.17.1 (Feb 2026), v1.17.0 (Feb 2026) — **biweekly releases**.

### Case Study 3: k9s — 32,953 stars

**Pattern:** Terminal-first K8s management tool

| What k9s Did | Frequency | Result |
|--------------|-----------|--------|
| Terminal UI experience | v0.50.18 (Jan 2026) | Developer delight factor |
| Demo videos | YouTube | Viral moments |
| Keyboard shortcuts | Vim-style | Developer adoption |
| Release notes | Humorous, detailed | Brand personality |
| GitHub Discussions | Active Q&A | Community engagement |

**Evidence:** [k9s GitHub repo](https://github.com/derailed/k9s) — 32,953 stars, 1,912 forks, created 2019-01-25. Release cadence: v0.50.18 (Jan 2026), v0.50.17 (Jan 2026) — **weekly releases during active development**.

### Case Study 4: in-toto-golang — 145 stars

**Pattern:** Standards framework (slower but steady growth)

| What in-toto Did | Frequency | Result |
|------------------|-----------|--------|
| RFC/specification first | ITE-10, ITE-11 | Standards legitimacy |
| Go implementation library | v0.10.0 (Jan 2026) | Multi-language support |
| Integration projects | argocd-interlace, signy | Ecosystem fit |
| Documentation | in-toto.io | Educational content |

**Evidence:** [in-toto-golang GitHub repo](https://github.com/in-toto/in-toto-golang) — 145 stars, created 2018-10-15. Release cadence: v0.10.0 (Jan 2026), v0.9.0 (May 2023) — **standards-driven, slower releases**.

### Case Study 5: GitHub CLI — 42,857 stars

**Pattern:** Official tool for existing platform

| What GitHub CLI Did | Frequency | Result |
|--------------------|-----------|--------|
| Official GitHub branding | GitHub CLI logo | Platform trust transfer |
| Installation docs | Homebrew, Snap, apt | Low friction |
| Extension system | gh extensions | Community plugins |
| Release notes | Detailed changelogs | Feature discovery |
| GitHub Actions integration | Native | Platform synergy |

**Evidence:** [GitHub CLI repo](https://github.com/cli/cli) — 42,857 stars, 4,061 forks, created 2019-10-03. Release cadence: **monthly releases**.

### Key Success Patterns Extracted

| Pattern | Implementation for SDP | Evidence |
|---------|------------------------|----------|
| **CLI-first entry point** | `sdp-evidence` CLI as flagship tool | k9s, GitHub CLI proven |
| **Standards-based** | in-toto attestation format (Phase 3) | OPA, in-toto proven |
| **CI/CD integration** | Evidence gate CI job for 5+ platforms | OPA, Kyverno proven |
| **Documentation excellence** | docs.sdp.dev with comprehensive guides | All top tools required |
| **Community hooks** | OpenCode, OhMyOpenCode, Beads adapters | Ecosystem adoption |
| **Release cadence** | Biweekly releases (v1.0.0 target at Week 8) | OPA, Kyverno proven |
| **Examples library** | 5+ language examples with GitHub Actions | Low barrier to entry |
| **Demo content** | Video walkthroughs of evidence validation | k9s virality factor |

---

## 12-Week Execution Plan

### Week 1-2: Foundation & Preparation

**Goal:** Ship v0.10.0 with CLI binary releases and comprehensive documentation

#### Week 1: CLI Release & Docs

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Complete `sdp-evidence` CLI binary packaging (Linux, macOS, Windows) | Maintainer | `sdp-evidence` v0.10.0 binaries |
| Tue | Write CLI installation documentation (Homebrew, apt, yum, Scoop, winget) | Maintainer | `docs/installation.md` |
| Wed | Create 5 Getting Started examples (Go, Python, TypeScript, Rust, Java) | Maintainer | `examples/getting-started/` directory |
| Thu | Set up docs.sdp.dev with Vercel/GitHub Pages | Maintainer | Live documentation site |
| Fri | Write v0.10.0 release notes with screenshots and video demo | Maintainer | GitHub Release v0.10.0 |
| Sat-Sun | Deploy CI integration examples (GitHub Actions, GitLab CI, CircleCI, Jenkins, Azure Pipelines) | Maintainer | 5 CI example repos |

**Milestones:**
- [ ] `sdp-evidence` CLI released with multi-platform binaries
- [ ] docs.sdp.dev live with complete installation guide
- [ ] 5 Getting Started examples published
- [ ] 5 CI integration examples published
- [ ] v0.10.0 release with video demo

**KPI Targets:**
- Stars: 16 → 25 (+9)
- CLI downloads: 0 → 20 (first week)
- Documentation coverage: ~60% → 80%

#### Week 2: GitHub Optimization & SEO

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Optimize README with clear value proposition, badges, and quick start | Maintainer | Updated `README.md` |
| Tue | Write GitHub Action for evidence validation (`sdp-evidence-action`) | Maintainer | `actions/sdp-evidence-action` repo |
| Wed | Create GitHub Topics: `evidence-validation`, `ai-agent-governance`, `supply-chain-security` | Maintainer | Repository tags |
| Thu | Write SEO-focused blog post: "Why Your AI Agents Need Evidence" | Maintainer | docs.sdp.dev/blog/why-evidence |
| Fri | Submit to Awesome-Go list, Awesome-CLI, Awesome-DevOps | Maintainer | PR submissions |
| Sat-Sun | Create GitHub Discussions templates (bug report, feature request, integration help) | Maintainer | `.github/DISCUSSION_TEMPLATE/` |

**Milestones:**
- [ ] README optimized with 5-sentence value proposition
- [ ] GitHub Action published and documented
- [ ] 3 awesome list submissions (Awesome-Go, Awesome-CLI, Awesome-DevOps)
- [ ] Blog post published on docs.sdp.dev
- [ ] GitHub Discussions templates created

**KPI Targets:**
- Stars: 25 → 35 (+10)
- GitHub Issues: 3 → 5 (+2 community issues)
- Blog views: 0 → 500 (first week)

---

### Week 3-4: Community Engagement & Integration

**Goal:** Ship v0.11.0 with OpenCode adapter and OhMyOpenCode integration

#### Week 3: OpenCode Integration

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Implement OpenCode session evidence emitter (`sdp-opencode-hook`) | Maintainer | `cmd/opencode-hook/` |
| Tue | Write OpenCode integration documentation | Maintainer | `docs/integrations/opencode.md` |
| Wed | Create demo video: "Validating Agent Evidence with OpenCode" | Maintainer | YouTube video (5 min) |
| Thu | Write blog post: "SDP × OpenCode: End-to-End Agent Governance" | Maintainer | docs.sdp.dev/blog/opencode-integration |
| Fri | Open issue on OpenCode repo: "SDP Evidence Integration PR" | Maintainer | OpenCode issue #XXX |
| Sat-Sun | Create OpenCode skill for SDP evidence validation | Maintainer | `.claude/skills/sdp-evidence.md` |

**Milestones:**
- [ ] OpenCode hook implemented and tested
- [ ] Demo video published on YouTube
- [ ] OpenCode integration documentation live
- [ ] OpenCode skill created and documented

**KPI Targets:**
- Stars: 35 → 50 (+15)
- YouTube video views: 0 → 200 (first week)
- OpenCode integration issue: Created

#### Week 4: v0.11.0 Release & Beads Integration

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Implement Beads graph adapter (`sdp-beads-bridge`) | Maintainer | `cmd/beads-bridge/` |
| Tue | Write Beads integration documentation | Maintainer | `docs/integrations/beads.md` |
| Wed | Test end-to-end: Issue → Beads → SDP evidence → PR | Maintainer | E2E test suite |
| Thu | Write v0.11.0 release notes (OpenCode + Beads integrations) | Maintainer | GitHub Release v0.11.0 |
| Fri | Submit to Hacker News: "SDP v0.11.0: OpenCode and Beads Integration" | Maintainer | HN submission |
| Sat-Sun | Create 5 Beads integration examples (feature, bugfix, refactor, test, doc) | Maintainer | `examples/beads-integration/` |

**Milestones:**
- [ ] Beads bridge implemented and tested
- [ ] E2E test suite passing (Issue → Beads → Evidence → PR)
- [ ] v0.11.0 released with integrations
- [ ] Hacker News submission live
- [ ] 5 Beads integration examples published

**KPI Targets:**
- Stars: 50 → 70 (+20)
- Hacker News upvotes: 30+ (target)
- E2E test pass rate: 100%

---

### Week 5-6: Standards Migration & Expansion

**Goal:** Ship v0.12.0 with in-toto attestation format (Phase 3 completion)

#### Week 5: in-toto Migration

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Define SDP predicate type: `https://sdp.dev/attestation/coding-workflow/v1` | Maintainer | `specs/predicate-type.json` |
| Tue | Rewrite `internal/evidence/` to use `github.com/in-toto/in-toto-golang` | Maintainer | Updated evidence package |
| Wed | Update `sdp-evidence` CLI to validate in-toto attestations | Maintainer | Updated CLI (v0.12.0) |
| Thu | Write migration guide for existing users (v0.11.0 → v0.12.0) | Maintainer | `docs/migration/v0.11-to-v0.12.md` |
| Fri | Submit PR to in-toto org: "Add SDP predicate to in-toto attestations list" | Maintainer | in-toto PR #XXX |
| Sat-Sun | Create Sigstore signing integration (keyless signing) | Maintainer | `cmd/sdp-sign/` |

**Milestones:**
- [ ] SDP predicate type defined and published
- [ ] Evidence package migrated to in-toto format
- [ ] `sdp-evidence` CLI updated for in-toto validation
- [ ] Migration guide published
- [ ] in-toto PR submitted

**KPI Targets:**
- Stars: 70 → 90 (+20)
- in-toto PR: Created
- Migration guide views: 100+ (first week)

#### Week 6: v0.12.0 Release & Standards Announcement

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Complete Sigstore signing integration (`sdp-sign`) | Maintainer | `sdp-sign` CLI |
| Tue | Write "SDP Goes Standards-Based: in-toto + Sigstore" blog post | Maintainer | docs.sdp.dev/blog/standards-pivot |
| Wed | Create demo video: "Signing Evidence with Sigstore Keyless Signing" | Maintainer | YouTube video (8 min) |
| Thu | Submit to r/golang, r/devops, r/kubernetes | Maintainer | Reddit posts |
| Fri | Write v0.12.0 release notes (in-toto + Sigstore) | Maintainer | GitHub Release v0.12.0 |
| Sat-Sun | Create 5 Sigstore integration examples (GitHub Actions, GitLab CI, CircleCI, Jenkins, Azure Pipelines) | Maintainer | `examples/sigstore-integration/` |

**Milestones:**
- [ ] `sdp-sign` CLI implemented and tested
- [ ] Blog post: "SDP Goes Standards-Based" published
- [ ] Demo video: "Signing Evidence with Sigstore" published
- [ ] 3 Reddit submissions (r/golang, r/devops, r/kubernetes)
- [ ] v0.12.0 released with in-toto + Sigstore
- [ ] 5 Sigstore integration examples published

**KPI Targets:**
- Stars: 90 → 120 (+30)
- Reddit upvotes: 50+ (combined)
- YouTube video views: 300+ (first week)

---

### Week 7-8: Launch Preparation & v1.0.0 RC

**Goal:** Ship v1.0.0-rc.1 with production-ready evidence layer

#### Week 7: Production Readiness

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Write production deployment guide (enterprise profile) | Maintainer | `docs/production/enterprise.md` |
| Tue | Create RBAC documentation for multi-tenant deployments | Maintainer | `docs/production/rbac.md` |
| Tue | Implement SIEM export (Splunk, Datadog, Elasticsearch) | Maintainer | `cmd/sdp-export/` |
| Wed | Write security audit report (third-party review summary) | Maintainer | `docs/security/audit-summary.md` |
| Thu | Create performance benchmarks (evidence validation throughput) | Maintainer | `docs/performance/benchmarks.md` |
| Fri | Write "Production-Ready SDP: Enterprise Guide" blog post | Maintainer | docs.sdp.dev/blog/production-guide |
| Sat-Sun | Create 3 enterprise deployment examples (single-tenant, multi-tenant, air-gapped) | Maintainer | `examples/production/` |

**Milestones:**
- [ ] Production deployment guide published
- [ ] RBAC documentation published
- [ ] SIEM export CLI implemented
- [ ] Security audit summary published
- [ ] Performance benchmarks published
- [ ] 3 enterprise deployment examples

**KPI Targets:**
- Stars: 120 → 150 (+30)
- Blog views: 500+ (first week)
- Enterprise guide views: 200+ (first week)

#### Week 8: v1.0.0-rc.1 Release

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Complete v1.0.0 feature freeze (no new features, bug fixes only) | Maintainer | v1.0.0-rc.1 branching |
| Tue | Write v1.0.0-rc.1 release notes (production-ready announcement) | Maintainer | GitHub Release v1.0.0-rc.1 |
| Wed | Create launch blog post: "SDP v1.0.0: Production-Ready Evidence Layer for AI Agents" | Maintainer | docs.sdp.dev/blog/v1.0.0-launch |
| Thu | Submit to Hacker News, r/golang, r/devops, r/kubernetes, r/MachineLearning | Maintainer | 5 launch submissions |
| Fri | Create launch video: "SDP v1.0.0 in 10 Minutes" | Maintainer | YouTube video (10 min) |
| Sat-Sun | Set up GitHub Sponsors page | Maintainer | GitHub Sponsors enabled |

**Milestones:**
- [ ] v1.0.0-rc.1 released
- [ ] Launch blog post published
- [ ] 5 launch submissions (HN, r/golang, r/devops, r/kubernetes, r/MachineLearning)
- [ ] Launch video published
- [ ] GitHub Sponsors enabled

**KPI Targets:**
- Stars: 150 → 200 (+50)
- Hacker News front page: Target
- Launch video views: 500+ (first week)
- GitHub Sponsors: First 5 supporters

---

### Week 9-10: Ecosystem Integration & Gas Town

**Goal:** Ship v1.0.0 with Gas Town adapter and 5 external integrations

#### Week 9: Gas Town Integration

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Implement Gas Town witness adapter (`sdp-gas-town-bridge`) | Maintainer | `cmd/gas-town-bridge/` |
| Tue | Write Gas Town integration documentation | Maintainer | `docs/integrations/gas-town.md` |
| Wed | Create demo: "SDP × Gas Town: Witness Monitoring for Agent CV" | Maintainer | Demo repo |
| Thu | Write blog post: "Gas Town + SDP: Agent Reputation with Evidence" | Maintainer | docs.sdp.dev/blog/gas-town-integration |
| Fri | Submit PR to Gas Town repo: "SDP Evidence Integration" | Maintainer | Gas Town PR #XXX |
| Sat-Sun | Create 3 Gas Town integration examples (baseline, advanced, enterprise) | Maintainer | `examples/gas-town-integration/` |

**Milestones:**
- [ ] Gas Town bridge implemented and tested
- [ ] Gas Town integration documentation published
- [ ] Demo repo created and published
- [ ] Gas Town PR submitted
- [ ] 3 Gas Town integration examples

**KPI Targets:**
- Stars: 200 → 240 (+40)
- Gas Town PR: Created
- Demo repo stars: 10+

#### Week 10: External Ecosystem Push

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Submit to OpenCode plugins list | Maintainer | OpenCode PR #XXX |
| Tue | Submit to OhMyOpenCode integrations list | Maintainer | OhMyOpenCode PR #XXX |
| Wed | Submit to Beads plugins list | Maintainer | Beads PR #XXX |
| Thu | Submit to Vibe Kanban integrations list | Maintainer | Vibe Kanban PR #XXX |
| Fri | Create ecosystem map visualization (Mermaid diagram) | Maintainer | `docs/ecosystem/map.md` |
| Sat-Sun | Write "SDP Ecosystem: Integrations Guide" blog post | Maintainer | docs.sdp.dev/blog/ecosystem-guide |

**Milestones:**
- [ ] 4 ecosystem integration PRs submitted (OpenCode, OhMyOpenCode, Beads, Vibe Kanban)
- [ ] Ecosystem map published
- [ ] Ecosystem guide blog post published

**KPI Targets:**
- Stars: 240 → 280 (+40)
- Ecosystem PRs: 4 created
- Ecosystem guide views: 300+ (first week)

---

### Week 11-12: v1.0.0 Launch & Future Roadmap

**Goal:** Ship v1.0.0 GA with complete evidence layer and publish future roadmap

#### Week 11: v1.0.0 GA Preparation

| Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Complete v1.0.0 bug fixes (no new features) | Maintainer | v1.0.0 final |
| Tue | Write v1.0.0 GA release notes (comprehensive changelog) | Maintainer | GitHub Release v1.0.0 |
| Wed | Create v1.0.0 launch video: "SDP v1.0.0: Evidence Layer for AI Agents" | Maintainer | YouTube video (15 min) |
| Thu | Write "SDP v1.0.0: The First Production-Ready Evidence Layer" blog post | Maintainer | docs.sdp.dev/blog/v1.0.0-ga |
| Fri | Submit to TechCrunch, InfoQ, The New Stack (outreach) | Maintainer | Media outreach emails |
| Sat-Sun | Create v1.0.0 launch checklist (binary releases, docs, examples, integrations) | Maintainer | `docs/releases/v1.0.0-checklist.md` |

**Milestones:**
- [ ] v1.0.0 GA released
- [ ] Launch video published
- [ ] Launch blog post published
- [ ] Media outreach emails sent
- [ ] v1.0.0 launch checklist completed

**KPI Targets:**
- Stars: 280 → 320 (+40)
- Launch video views: 1000+ (first week)
- Media mentions: 1+ (target)

#### Week 12: Future Roadmap & Community

**Day | Tasks | Owner | Artifacts |
|-----|-------|-------|-----------|
| Mon | Write "SDP Roadmap: Beyond v1.0.0" (Phase 8-9 K8s pipeline) | Maintainer | `docs/roadmap/beyond-v1.0.0.md` |
| Tue | Create community contribution guide (CONTRIBUTING.md revamp) | Maintainer | `CONTRIBUTING.md` |
| Wed | Set up GitHub Discussions categories (General, Support, Feature Requests, Integrations) | Maintainer | GitHub Discussions configured |
| Thu | Write "Building SDP Together: Community Guide" blog post | Maintainer | docs.sdp.dev/blog/community-guide |
| Fri | Create weekly sync schedule (community office hours) | Maintainer | Calendar invite + README badge |
| Sat-Sun | Write 12-week retrospective (what worked, what didn't, lessons learned) | Maintainer | `docs/plans/12-week-retrospective.md` |

**Milestones:**
- [ ] Beyond v1.0.0 roadmap published
- [ ] Contribution guide published
- [ ] GitHub Discussions configured
- [ ] Community guide published
- [ ] Weekly sync schedule established
- [ ] 12-week retrospective published

**KPI Targets:**
- Stars: 320 → 350 (+30)
- GitHub Discussions: 20+ threads active
- Weekly sync attendance: 10+ (first session)
- 12-week retrospective: Published

---

## Channel Strategy

### Primary Channels (80% of effort)

| Channel | Frequency | Format | Owner | KPI |
|---------|-----------|--------|-------|-----|
| **GitHub Releases** | Biweekly | Binary releases + changelog | Maintainer | 200+ downloads/release |
| **GitHub Actions Marketplace** | Week 2 | `sdp-evidence-action` | Maintainer | 50+ installs/month |
| **docs.sdp.dev Blog** | Weekly | Technical tutorials | Maintainer | 500+ views/post |
| **YouTube** | Biweekly | Demo videos (5-15 min) | Maintainer | 300+ views/video |
| **Hacker News** | Key milestones | Launch announcements | Maintainer | Front page (30+ upvotes) |
| **Reddit (r/golang, r/devops, r/kubernetes)** | Biweekly | Integration updates | Maintainer | 50+ upvotes/post |

### Secondary Channels (15% of effort)

| Channel | Frequency | Format | Owner | KPI |
|---------|-----------|--------|-------|-----|
| **Twitter/X** | Daily | Tips, releases, integrations | Maintainer | 1,000+ followers |
| **LinkedIn** | Weekly | Enterprise use cases | Maintainer | 500+ followers |
| **Discord/Slack** | Daily | Community support | Maintainer | 100+ members |
| **Medium/Dev.to** | Weekly | Technical deep dives | Maintainer | 200+ views/post |
| **Conferences** | Quarterly | Talks, workshops | Maintainer | 1+ talk accepted |

### Ecosystem Channels (5% of effort)

| Channel | Frequency | Format | Owner | KPI |
|---------|-----------|--------|-------|-----|
| **OpenCode Discord** | Weekly | Integration updates | Maintainer | 50+ mentions |
| **Beads Discussions** | Weekly | Feature sync | Maintainer | 10+ threads |
| **OhMyOpenCode Issues** | Weekly | PR follow-ups | Maintainer | 5+ PRs merged |
| **Gas Town Discord** | Weekly | Witness integration | Maintainer | 20+ mentions |
| **Vibe Kanban Issues** | Weekly | Orchestration sync | Maintainer | 3+ PRs merged |

---

## Measurable Funnel Metrics

### Funnel Definition

```
Awareness (stars) → Interest (clones) → Evaluation (issues) → Adoption (downloads) → Advocacy (forks)
```

### Metric Targets by Week

| Week | Stars (Awareness) | Clones/week (Interest) | Open Issues (Evaluation) | Downloads/month (Adoption) | Forks (Advocacy) |
|------|------------------|----------------------|---------------------------|---------------------------|------------------|
| Start | 16 | - | 3 | 0 | 0 |
| 2 | 25 | 50 | 5 | 20 | 1 |
| 4 | 50 | 100 | 8 | 50 | 5 |
| 6 | 90 | 200 | 12 | 100 | 10 |
| 8 | 150 | 400 | 15 | 150 | 20 |
| 10 | 240 | 600 | 18 | 200 | 30 |
| 12 | 350 | 800 | 20 | 300 | 50 |

### Conversion Rate Targets

| Funnel Stage | Conversion Rate | Target |
|--------------|-----------------|--------|
| Awareness → Interest | Clones/Stars | 200% (each star generates 2 clones/month) |
| Interest → Evaluation | Issues/Clones | 5% (5% of cloners open issues) |
| Evaluation → Adoption | Downloads/Issues | 1,500% (each issue drives 15 downloads) |
| Adoption → Advocacy | Forks/Downloads | 15% (15% of downloaders fork) |

### Metric Tracking

**GitHub API Metrics:**
```bash
# Stars
gh api repos/fall-out-bug/sdp --jq '{stars: .stargazers_count, forks: .forks_count}'

# Traffic (clones, views)
gh api repos/fall-out-bug/sdp/traffic/clones
gh api repos/fall-out-bug/sdp/traffic/views

# Issues
gh api repos/fall-out-bug/sdp/issues --jq 'length'

# Downloads (from GitHub releases - requires manual tracking or Google Analytics on docs)
# Use docs.sdp.dev analytics (Vercel Analytics / Plausible / Google Analytics)
```

**Weekly Metric Report Template:**
```markdown
## Week X Metrics (YYYY-MM-DD)

| Metric | Previous | Current | Change | Target |
|--------|----------|---------|--------|--------|
| Stars | XX | XX | +X | XX |
| Forks | XX | XX | +X | XX |
| Clones/week | XX | XX | +X | XX |
| Open Issues | XX | XX | +X | XX |
| Downloads/month | XX | XX | +X | XX |
| YouTube views (last video) | XX | XX | +X | XX |
| Blog views (last post) | XX | XX | +X | XX |

### Highlights
- [ ] Key achievement 1
- [ ] Key achievement 2

### Blockers
- [ ] Blocker 1
- [ ] Blocker 2

### Next Week Focus
- [ ] Focus area 1
- [ ] Focus area 2
```

---

## Release Cadence

### Release Schedule

| Week | Version | Type | Scope | Release Notes |
|------|---------|------|-------|---------------|
| 1 | v0.10.0 | Minor | CLI binary releases + docs | `docs/releases/v0.10.0.md` |
| 2 | - | Patch | Bug fixes from v0.10.0 | `docs/releases/v0.10.1.md` |
| 4 | v0.11.0 | Minor | OpenCode + Beads integrations | `docs/releases/v0.11.0.md` |
| 5 | - | Patch | Bug fixes from v0.11.0 | `docs/releases/v0.11.1.md` |
| 6 | v0.12.0 | Major | in-toto + Sigstore migration | `docs/releases/v0.12.0.md` |
| 7 | - | Patch | Bug fixes from v0.12.0 | `docs/releases/v0.12.1.md` |
| 8 | v1.0.0-rc.1 | RC | Production-ready feature freeze | `docs/releases/v1.0.0-rc.1.md` |
| 9 | v1.0.0-rc.2 | RC | Bug fixes from RC1 | `docs/releases/v1.0.0-rc.2.md` |
| 11 | v1.0.0 | GA | Production-ready release | `docs/releases/v1.0.0.md` |
| 12 | v1.1.0 | Minor | Post-launch improvements | `docs/releases/v1.1.0.md` |

### Release Cadence Pattern

**Biweekly releases** (every 2 weeks):
- Minor version releases (v0.X.0) at weeks 1, 4, 6
- RC releases (v1.0.0-rc.X) at weeks 8, 9
- GA release (v1.0.0) at week 11
- Post-launch (v1.X.0) at week 12

**Patch releases** (as needed, between minor releases):
- Critical bug fixes within 24 hours
- Non-critical bug fixes in next biweekly release

### Release Process Checklist

**Before Release:**
- [ ] All tests passing (`go test ./...`)
- [ ] Documentation updated (release notes, migration guide if needed)
- [ ] Binary packages built (Linux, macOS, Windows)
- [ ] Changelog reviewed and verified
- [ ] Breaking changes documented (if any)
- [ ] Examples tested and updated (if needed)

**Release Day:**
- [ ] Create GitHub release with:
  - Release notes (markdown)
  - Binary attachments (Linux, macOS, Windows)
  - Checksums (SHA256)
  - Signature (GPG/Sigstore)
  - Video demo (link to YouTube)
- [ ] Update docs.sdp.dev with new version
- [ ] Publish blog post (if major release)
- [ ] Submit to Hacker News (if major release)
- [ ] Post to Discord/Slack/Reddit

**After Release:**
- [ ] Monitor GitHub issues for bug reports
- [ ] Respond to community questions (GitHub Discussions, Discord, Reddit)
- [ ] Track download metrics (GitHub releases, docs analytics)
- [ ] Plan next release based on feedback

---

## Content Cadence

### Blog Post Schedule

| Week | Topic | Type | Target Views |
|------|-------|------|-------------|
| 1 | "Why Your AI Agents Need Evidence" | Problem statement | 500 |
| 2 | "SDP CLI Quick Start" | Tutorial | 500 |
| 3 | "SDP × OpenCode: End-to-End Agent Governance" | Integration | 300 |
| 4 | "SDP v0.11.0: OpenCode and Beads Integration" | Release announcement | 500 |
| 5 | "Migrating to in-toto: SDP v0.12.0 Guide" | Migration guide | 200 |
| 6 | "SDP Goes Standards-Based: in-toto + Sigstore" | Standards announcement | 500 |
| 7 | "Production-Ready SDP: Enterprise Guide" | Enterprise focus | 500 |
| 8 | "SDP v1.0.0-rc.1: Production-Ready Evidence Layer" | RC announcement | 500 |
| 9 | "Gas Town + SDP: Agent Reputation with Evidence" | Integration | 300 |
| 10 | "SDP Ecosystem: Integrations Guide" | Ecosystem | 300 |
| 11 | "SDP v1.0.0: The First Production-Ready Evidence Layer" | GA announcement | 1000 |
| 12 | "Building SDP Together: Community Guide" | Community | 300 |

### Video Schedule

| Week | Topic | Duration | Target Views |
|------|-------|----------|-------------|
| 1 | "Validating Evidence with sdp-evidence CLI" | 5 min | 200 |
| 3 | "Validating Agent Evidence with OpenCode" | 5 min | 200 |
| 6 | "Signing Evidence with Sigstore Keyless Signing" | 8 min | 300 |
| 8 | "SDP v1.0.0 in 10 Minutes" | 10 min | 500 |
| 11 | "SDP v1.0.0: Evidence Layer for AI Agents" | 15 min | 1000 |

### Social Media Schedule

| Platform | Frequency | Content Type | KPI |
|----------|-----------|--------------|-----|
| Twitter/X | Daily (Mon-Fri) | Tips, releases, integrations | 1,000+ followers |
| LinkedIn | Weekly | Enterprise use cases | 500+ followers |
| Reddit | Biweekly | Integration updates | 50+ upvotes/post |
| Hacker News | Key milestones | Launch announcements | Front page (30+ upvotes) |

---

## Artifact Checklist

### Week 1-2 Artifacts

- [ ] `sdp-evidence` CLI v0.10.0 (Linux, macOS, Windows binaries)
- [ ] `docs/installation.md` (Homebrew, apt, yum, Scoop, winget)
- [ ] `examples/getting-started/` (5 languages: Go, Python, TypeScript, Rust, Java)
- [ ] `examples/ci-integration/` (5 CI platforms: GitHub Actions, GitLab CI, CircleCI, Jenkins, Azure Pipelines)
- [ ] `docs.sdp.dev` (live documentation site)
- [ ] `README.md` (optimized with value proposition, badges, quick start)
- [ ] `actions/sdp-evidence-action` (GitHub Action)
- [ ] `docs/blog/why-evidence.md` (blog post)
- [ ] `.github/DISCUSSION_TEMPLATE/` (3 templates: bug report, feature request, integration help)

### Week 3-4 Artifacts

- [ ] `cmd/opencode-hook/` (OpenCode session evidence emitter)
- [ ] `docs/integrations/opencode.md` (OpenCode integration documentation)
- [ ] YouTube video: "Validating Agent Evidence with OpenCode" (5 min)
- [ ] `docs/blog/opencode-integration.md` (blog post)
- [ ] `.claude/skills/sdp-evidence.md` (OpenCode skill)
- [ ] `cmd/beads-bridge/` (Beads graph adapter)
- [ ] `docs/integrations/beads.md` (Beads integration documentation)
- [ ] E2E test suite (Issue → Beads → SDP evidence → PR)
- [ ] GitHub Release v0.11.0 (OpenCode + Beads integrations)
- [ ] `examples/beads-integration/` (5 examples: feature, bugfix, refactor, test, doc)

### Week 5-6 Artifacts

- [ ] `specs/predicate-type.json` (SDP predicate type: `https://sdp.dev/attestation/coding-workflow/v1`)
- [ ] `internal/evidence/` (rewritten for in-toto-golang)
- [ ] `sdp-evidence` CLI v0.12.0 (in-toto validation)
- [ ] `docs/migration/v0.11-to-v0.12.md` (migration guide)
- [ ] `cmd/sdp-sign/` (Sigstore signing integration)
- [ ] `docs/blog/standards-pivot.md` (blog post)
- [ ] YouTube video: "Signing Evidence with Sigstore Keyless Signing" (8 min)
- [ ] `examples/sigstore-integration/` (5 CI platforms)
- [ ] GitHub Release v0.12.0 (in-toto + Sigstore)
- [ ] in-toto PR (submitted to in-toto org)

### Week 7-8 Artifacts

- [ ] `docs/production/enterprise.md` (production deployment guide)
- [ ] `docs/production/rbac.md` (RBAC documentation)
- [ ] `cmd/sdp-export/` (SIEM export: Splunk, Datadog, Elasticsearch)
- [ ] `docs/security/audit-summary.md` (security audit report)
- [ ] `docs/performance/benchmarks.md` (performance benchmarks)
- [ ] `docs/blog/production-guide.md` (blog post)
- [ ] `examples/production/` (3 examples: single-tenant, multi-tenant, air-gapped)
- [ ] GitHub Release v1.0.0-rc.1 (production-ready)
- [ ] `docs/blog/v1.0.0-launch.md` (launch blog post)
- [ ] YouTube video: "SDP v1.0.0 in 10 Minutes" (10 min)
- [ ] GitHub Sponsors page (enabled)

### Week 9-10 Artifacts

- [ ] `cmd/gas-town-bridge/` (Gas Town witness adapter)
- [ ] `docs/integrations/gas-town.md` (Gas Town integration documentation)
- [ ] Demo repo: "SDP × Gas Town: Witness Monitoring for Agent CV"
- [ ] `docs/blog/gas-town-integration.md` (blog post)
- [ ] Gas Town PR (submitted to Gas Town repo)
- [ ] `examples/gas-town-integration/` (3 examples: baseline, advanced, enterprise)
- [ ] OpenCode PR (submitted to OpenCode plugins list)
- [ ] OhMyOpenCode PR (submitted to OhMyOpenCode integrations list)
- [ ] Beads PR (submitted to Beads plugins list)
- [ ] Vibe Kanban PR (submitted to Vibe Kanban integrations list)
- [ ] `docs/ecosystem/map.md` (ecosystem map visualization)
- [ ] `docs/blog/ecosystem-guide.md` (ecosystem guide blog post)

### Week 11-12 Artifacts

- [ ] GitHub Release v1.0.0 (GA)
- [ ] `docs/releases/v1.0.0.md` (comprehensive changelog)
- [ ] YouTube video: "SDP v1.0.0: Evidence Layer for AI Agents" (15 min)
- [ ] `docs/blog/v1.0.0-ga.md` (GA blog post)
- [ ] Media outreach emails (TechCrunch, InfoQ, The New Stack)
- [ ] `docs/releases/v1.0.0-checklist.md` (v1.0.0 launch checklist)
- [ ] `docs/roadmap/beyond-v1.0.0.md` (future roadmap)
- [ ] `CONTRIBUTING.md` (revamped community contribution guide)
- [ ] GitHub Discussions (configured with categories)
- [ ] `docs/blog/community-guide.md` (community guide blog post)
- [ ] Weekly sync calendar (README badge)
- [ ] `docs/plans/12-week-retrospective.md` (12-week retrospective)

---

## KPI Tracking Dashboard

### Weekly KPI Template

Create a markdown file `docs/plans/kpi-week-X.md` for each week:

```markdown
# Week X KPI Report (YYYY-MM-DD to YYYY-MM-DD)

## Funnel Metrics

| Metric | Previous | Current | Change | Target | % of Target |
|--------|----------|---------|--------|--------|-------------|
| Stars (Awareness) | XX | XX | +X | XX | XX% |
| Forks (Advocacy) | XX | XX | +X | XX | XX% |
| Clones/week (Interest) | XX | XX | +X | XX | XX% |
| Open Issues (Evaluation) | XX | XX | +X | XX | XX% |
| Downloads/month (Adoption) | XX | XX | +X | XX | XX% |

## Channel Metrics

| Channel | Metric | Current | Target | % of Target |
|---------|--------|---------|--------|-------------|
| GitHub Releases | Downloads this release | XX | XX | XX% |
| docs.sdp.dev Blog | Views (last post) | XX | XX | XX% |
| YouTube | Views (last video) | XX | XX | XX% |
| Twitter/X | Followers | XX | XX | XX% |
| Reddit | Upvotes (last post) | XX | XX | XX% |
| Hacker News | Upvotes | XX | XX | XX% |

## Content Metrics

| Content Type | Published | Views | Downloads | Comments |
|--------------|-----------|-------|------------|-----------|
| Blog Post | Title (link TBD) | XX | - | XX |
| Video | Title (link TBD) | XX | - | XX |
| GitHub Release | v0.XX.X | - | XX | - |
| Example/Integration | Name (link TBD) | - | XX | XX |

## Milestones

- [ ] Milestone 1: Description
- [ ] Milestone 2: Description

## Blockers

- [ ] Blocker 1: Description
  - Impact: High/Medium/Low
  - Owner: @username
  - ETA: YYYY-MM-DD

- [ ] Blocker 2: Description
  - Impact: High/Medium/Low
  - Owner: @username
  - ETA: YYYY-MM-DD

## Highlights

- Highlight 1: What went well this week
- Highlight 2: Community engagement win
- Highlight 3: Technical achievement

## Lessons Learned

- Lesson 1: What we'd do differently
- Lesson 2: What we'll repeat next week

## Next Week Focus

- Focus area 1: Description
- Focus area 2: Description
- Focus area 3: Description

## Tasks for Next Week

- [ ] Task 1: Description (@username)
- [ ] Task 2: Description (@username)
- [ ] Task 3: Description (@username)
```

### 12-Week Summary Dashboard

Create `docs/plans/kpi-12week-summary.md`:

```markdown
# 12-Week Growth Summary (YYYY-MM-DD to YYYY-MM-DD)

## Funnel Metrics - 12 Week Trajectory

```
Stars:   16 → 350 (+334, +2,088%)
Forks:   0  → 50  (+50,  +100%)
Clones:  -  → 800/week
Issues:  3  → 20  (+17,  +567%)
Downloads: 0 → 300/month
```

## Channel Performance

| Channel | Target | Actual | % of Target | Best Week |
|---------|--------|--------|-------------|-----------|
| GitHub Stars | 350 | XX | XX% | Week XX |
| Forks | 50 | XX | XX% | Week XX |
| Blog Views (avg) | 500/post | XX | XX% | Week XX |
| YouTube Views (avg) | 300/video | XX | XX% | Week XX |
| Hacker News Front Page | 2 | XX | XX% | Week XX |

## Milestone Completion

| Milestone | Planned Week | Completed Week | Status |
|-----------|--------------|----------------|--------|
| v0.10.0 CLI Release | 1 | XX | ✅/❌ |
| OpenCode Integration | 3 | XX | ✅/❌ |
| v0.11.0 Release | 4 | XX | ✅/❌ |
| in-toto Migration | 5 | XX | ✅/❌ |
| v0.12.0 Release | 6 | XX | ✅/❌ |
| v1.0.0-rc.1 Release | 8 | XX | ✅/❌ |
| Gas Town Integration | 9 | XX | ✅/❌ |
| Ecosystem PRs | 10 | XX | ✅/❌ |
| v1.0.0 GA Release | 11 | XX | ✅/❌ |
| Community Guide | 12 | XX | ✅/❌ |

## Top Performing Content

| Content | Views | Downloads | Comments | Stars/Forks Generated |
|---------|-------|------------|-----------|----------------------|
| Blog Post: [Title] | XXX | - | XX | +XX stars |
| Video: [Title] | XXX | - | XX | +XX stars |
| GitHub Release: v0.XX.X | - | XXX | - | +XX downloads |
| Integration: [Name] | - | XXX | XX | +XX forks |

## Community Engagement

| Metric | Start | End | Change |
|--------|-------|-----|--------|
| GitHub Issues | 3 | XX | +XX |
| GitHub PRs | 0 | XX | +XX |
| GitHub Discussions | 0 | XX | +XX |
| External PRs (ecosystem) | 0 | XX | +XX |
| Twitter/X Followers | 0 | XX | +XX |
| Discord/Slack Members | 0 | XX | +XX |

## Lessons Learned

### What Worked
1. [Success pattern 1]
2. [Success pattern 2]
3. [Success pattern 3]

### What Didn't Work
1. [Failure pattern 1]
2. [Failure pattern 2]

### What We'd Do Differently
1. [Adjustment 1]
2. [Adjustment 2]

## Recommendations for Next 12 Weeks

1. [Recommendation 1]
2. [Recommendation 2]
3. [Recommendation 3]

## Appendix: Weekly KPI Reports

- Week 1 (`kpi-week-1.md`)
- Week 2 (`kpi-week-2.md`)
- Week 3 (`kpi-week-3.md`)
- Week 4 (`kpi-week-4.md`)
- Week 5 (`kpi-week-5.md`)
- Week 6 (`kpi-week-6.md`)
- Week 7 (`kpi-week-7.md`)
- Week 8 (`kpi-week-8.md`)
- Week 9 (`kpi-week-9.md`)
- Week 10 (`kpi-week-10.md`)
- Week 11 (`kpi-week-11.md`)
- Week 12 (`kpi-week-12.md`)
```

---

## Critical Success Factors

### Factor 1: Weekly Shipping Discipline

**Rule:** Ship something every week — even if it's just a patch release, blog post, or example.

**Enforcement:**
- Monday morning review: What did we ship last week? What are we shipping this week?
- Friday EOD review: Did we ship what we planned? If no, why?
- Public accountability: GitHub Releases, blog posts, social media posts

**Evidence:** Successful tools (OPA, Kyverno, k9s) shipped **monthly or weekly** during growth phases. SDP has already proven this with 10 releases in 2 months — maintain this cadence.

### Factor 2: Standards-Based Approach

**Rule:** Every decision should reference an existing standard or contribute to one.

**Standards to leverage:**
- **in-toto** — Evidence envelope format (Phase 3)
- **Sigstore** — Keyless signing infrastructure (Phase 4)
- **OPA/Rego** — Policy-as-code (Phase 5)
- **SLSA** — Supply chain levels (compliance)

**Evidence:** in-toto-golang (145 stars), OPA (11,292 stars) achieved growth through standards legitimacy. SDP is already committed to in-toto migration (Phase 3).

### Factor 3: CI/CD Integration as Primary Use Case

**Rule:** The evidence gate is the primary value proposition. Make it dead simple to add to CI.

**Implementation:**
- GitHub Action (`sdp-evidence-action`) — Week 2
- 5 CI integration examples (GitHub Actions, GitLab CI, CircleCI, Jenkins, Azure Pipelines) — Week 1
- CI job template: Copy-paste into `.github/workflows/evidence-gate.yml`

**Evidence:** OPA (11,292 stars) and Kyverno (7,453 stars) achieved enterprise adoption through CI/CD integration. GitHub CLI (42,857 stars) achieved platform dominance through native GitHub Actions.

### Factor 4: Documentation Excellence

**Rule:** If it's not documented, it doesn't exist.

**Documentation targets:**
- docs.sdp.dev (live site) — Week 1
- 90% documentation coverage — Week 12
- 5 Getting Started examples (5 languages) — Week 1
- 5 CI integration examples (5 platforms) — Week 1
- Migration guides (v0.11.0 → v0.12.0) — Week 5
- Production deployment guide (enterprise) — Week 7

**Evidence:** All top tools (OPA, Kyverno, k9s, in-toto) invested heavily in documentation. docs.sdp.dev is already planned for Week 1.

### Factor 5: Community Hooks

**Rule:** Make it easy for other tools to integrate with SDP. Every integration is a growth vector.

**Integrations to ship:**
- OpenCode (session evidence emitter) — Week 3
- Beads (graph adapter) — Week 4
- OhMyOpenCode (permission → guard bridge) — Week 10
- Gas Town (witness adapter) — Week 9
- Vibe Kanban (orchestration sync) — Week 10

**Evidence:** OPA (11,292 stars) achieved ecosystem growth through policy libraries and integrations. Kyverno (7,453 stars) achieved adoption through K8s ecosystem fit.

### Factor 6: Demo Content

**Rule:** If you can't show it working in 5 minutes, it doesn't exist.

**Demo schedule:**
- Week 1: "Validating Evidence with sdp-evidence CLI" (5 min)
- Week 3: "Validating Agent Evidence with OpenCode" (5 min)
- Week 6: "Signing Evidence with Sigstore Keyless Signing" (8 min)
- Week 8: "SDP v1.0.0 in 10 Minutes" (10 min)
- Week 11: "SDP v1.0.0: Evidence Layer for AI Agents" (15 min)

**Evidence:** k9s (32,953 stars) achieved viral moments through terminal UI demo videos. GitHub CLI (42,857 stars) achieved platform adoption through official tutorials.

### Factor 7: Launch Discipline

**Rule:** Every major release is a launch. Treat it as such.

**Launch checklist:**
- Release notes (markdown)
- Binary packages (Linux, macOS, Windows)
- Checksums (SHA256)
- Signature (GPG/Sigstore)
- Video demo (YouTube)
- Blog post (docs.sdp.dev)
- Hacker News submission
- Reddit submissions (r/golang, r/devops, r/kubernetes)
- Social media posts (Twitter, LinkedIn)
- Discord/Slack announcements

**Evidence:** OPA, Kyverno, k9s all follow disciplined launch processes with comprehensive release notes and demo content.

---

## Risk Mitigation

### Risk 1: Low Engagement (Stars, Forks, Issues)

**Probability:** Medium  
**Impact:** High  
**Mitigation:**
- Focus on **community hooks** (OpenCode, Beads, OhMyOpenCode, Gas Town) — every integration is a growth vector
- Submit to **awesome lists** (Awesome-Go, Awesome-CLI, Awesome-DevOps) — organic discovery
- Engage in **Hacker News** and **Reddit** communities — target r/golang, r/devops, r/kubernetes
- **Weekly shipping discipline** — consistent activity signals project health

**Contingency:** If Week 6 stars < 100, increase Hacker News and Reddit frequency to weekly (instead of biweekly).

### Risk 2: Integration PRs Not Merged

**Probability:** Medium  
**Impact:** Medium  
**Mitigation:**
- **Start early** — submit PRs by Week 9-10 (before v1.0.0 GA)
- **Low barrier to entry** — keep integrations minimal (hooks, adapters)
- **Active maintenance** — respond to feedback within 24 hours
- **Community engagement** — participate in Discord/Slack discussions

**Contingency:** If 3+ ecosystem PRs are not merged by Week 10, ship integrations as separate repos (e.g., `github.com/sdp-dev/opencode-integration`).

### Risk 3: Release Cadence Slips

**Probability:** Low  
**Impact:** Medium  
**Mitigation:**
- **Biweekly release cadence** — only major releases, no scope creep
- **RC releases** — v1.0.0-rc.1 at Week 8, v1.0.0-rc.2 at Week 9
- **Feature freeze** — v1.0.0-rc.1 is feature freeze, only bug fixes
- **Patch releases** — ship critical bugs within 24 hours

**Contingency:** If Week 6 v0.12.0 is delayed, skip v0.11.0 and ship v0.12.0 directly (in-toto migration is higher priority).

### Risk 4: Documentation Quality

**Probability:** Medium  
**Impact:** Medium  
**Mitigation:**
- **docs.sdp.dev** — professional documentation site (Week 1)
- **Technical writers** — allocate time for documentation (20% of weekly effort)
- **Community feedback** — GitHub Discussions for documentation questions
- **90% coverage target** — explicit KPI for Week 12

**Contingency:** If Week 4 documentation coverage < 70%, allocate additional time (40% of weekly effort) until Week 6.

### Risk 5: Burnout

**Probability:** High  
**Impact:** High  
**Mitigation:**
- **Realistic scope** — 12-week plan is aggressive but achievable
- **Weekly review** — cancel/defer tasks if blockers emerge
- **Community help** - accept PRs from community (if any)
- **Quality over quantity** — it's better to ship 1 high-quality blog post than 3 low-quality ones

**Contingency:** If Week 8 burnout symptoms appear, reduce cadence to **monthly releases** (v1.0.0-rc.1, v1.0.0, v1.1.0).

---

## Success Criteria

### Absolute Success

After 12 weeks, SDP is a **production-ready evidence layer** for AI coding agents with:

| Metric | Target |
|--------|--------|
| Stars | 500+ |
| Forks | 50+ |
| CLI downloads | 200+/month |
| GitHub Issues | 20+ open |
| Ecosystem PRs | 3+ merged |
| Documentation coverage | 90%+ |
| CI integration examples | 5+ platforms |
| Language examples | 5+ languages |
| Blog posts | 12 published |
| Demo videos | 5 published |

### Minimum Success

After 12 weeks, SDP is a **usable evidence layer** for AI coding agents with:

| Metric | Target |
|--------|--------|
| Stars | 200+ |
| Forks | 20+ |
| CLI downloads | 100+/month |
| GitHub Issues | 10+ open |
| Ecosystem PRs | 1+ merged |
| Documentation coverage | 70%+ |
| CI integration examples | 3+ platforms |
| Language examples | 3+ languages |
| Blog posts | 8 published |
| Demo videos | 3 published |

---

## Appendix A: Weekly Task Templates

### Week X Task Template

```markdown
# Week X Tasks (YYYY-MM-DD to YYYY-MM-DD)

## Priority 1 (Must Ship)

- [ ] Task 1: Description (@username) — Evidence (link TBD) — Deadline (YYYY-MM-DD)
- [ ] Task 2: Description (@username) — Evidence (link TBD) — Deadline (YYYY-MM-DD)
- [ ] Task 3: Description (@username) — Evidence (link TBD) — Deadline (YYYY-MM-DD)

## Priority 2 (Should Ship)

- [ ] Task 4: Description (@username) — Evidence (link TBD) — Deadline (YYYY-MM-DD)
- [ ] Task 5: Description (@username) — Evidence (link TBD) — Deadline (YYYY-MM-DD)

## Priority 3 (Nice to Ship)

- [ ] Task 6: Description (@username) — Evidence (link TBD) — Deadline (YYYY-MM-DD)

## Blockers

- [ ] Blocker 1: Description (@username, ETA: YYYY-MM-DD)
- [ ] Blocker 2: Description (@username, ETA: YYYY-MM-DD)

## Deliverables

- [ ] Deliverable 1: Link (url TBD) — Status (red/yellow/green)
- [ ] Deliverable 2: Link (url TBD) — Status (red/yellow/green)
- [ ] Deliverable 3: Link (url TBD) — Status (red/yellow/green)

## This Week's Shipment

**Release:** v0.XX.X  
**Release Notes:** Link (url TBD)  
**Binary Downloads:** Linux (url TBD), macOS (url TBD), Windows (url TBD)  
**Blog Post:** Title (url TBD)  
**Video:** Title (url TBD)  
**Examples:** Link (url TBD)

## Next Week Preview

**Planned Release:** v0.XX.X  
**Key Features:** Feature 1, Feature 2, Feature 3  
**Planned Blog Post:** Title (url TBD)  
**Planned Video:** Title (url TBD)
```

---

## Appendix B: Communication Templates

### Hacker News Submission Template

```
Title: SDP v0.XX.X: [Feature Summary]

Description:
SDP is an evidence layer for AI coding agents. It proves that agents followed a process (intent → plan → execute → verify → review) with machine-readable attestations.

What's new in v0.XX.X:
- [Feature 1]
- [Feature 2]
- [Feature 3]

Key features:
- CLI tool: `sdp-evidence validate` — validate evidence envelopes in 1 command
- CI gate: Evidence validation blocks PR merge if evidence is incomplete or invalid
- Standards-based: in-toto attestation format + Sigstore signing
- Ecosystem: Integrations with OpenCode, Beads, OhMyOpenCode, Gas Town, Vibe Kanban

GitHub: https://github.com/fall-out-bug/sdp
Docs: https://docs.sdp.dev
Install: `go install github.com/fall-out-bug/sdp/cmd/sdp-evidence@latest`
```

### Reddit Submission Template (r/golang)

```
Title: [Go] SDP v0.XX.X: Evidence layer for AI coding agents (in-toto + Sigstore)

Description:
SDP is a Go-based evidence layer for AI coding agents. It proves that agents followed a process (intent → plan → execute → verify → review) with machine-readable attestations.

v0.XX.X highlights:
- [Feature 1] — Go implementation detail
- [Feature 2] — Go implementation detail
- [Feature 3] — Go implementation detail

Key Go features:
- CLI tool: `sdp-evidence validate` — Cobra-based CLI with subcommands
- in-toto integration: `github.com/in-toto/in-toto-golang` for attestation validation
- Sigstore signing: `github.com/sigstore/cosign/v2` for keyless signing
- CI integration: GitHub Action for evidence validation

GitHub: https://github.com/fall-out-bug/sdp
Docs: https://docs.sdp.dev
Install: `go install github.com/fall-out-bug/sdp/cmd/sdp-evidence@latest`

Feedback welcome! Looking for Go developers interested in AI agent governance.
```

### Blog Post Template

```markdown
# [Blog Post Title]

## Summary

[2-3 sentence summary of what this post covers and why it matters]

## Background

[Context: What problem are we solving? Who is this for?]

## The Solution

[Technical details: How does SDP solve this problem? Code examples]

## Example

[Walkthrough example: Step-by-step tutorial with code snippets]

## Next Steps

[Call to action: Try it out, provide feedback, contribute]

## Resources

- [GitHub repo](https://github.com/fall-out-bug/sdp)
- [Documentation](https://docs.sdp.dev)
- [Installation guide](https://docs.sdp.dev/installation)
- Discord/Slack (link TBD)
```

---

## Appendix C: Release Notes Template

```markdown
# SDP v0.XX.X Release Notes

**Release Date:** YYYY-MM-DD  
**Stars:** [Current]  
**Downloads:** [Total]

## What's New

### Feature 1

[Description of feature 1]

```bash
# Example usage
sdp-evidence validate --evidence .sdp/evidence/run-123.json
```

### Feature 2

[Description of feature 2]

### Feature 3

[Description of feature 3]

## Improvements

- [Improvement 1]
- [Improvement 2]

## Bug Fixes

- [Bug fix 1]
- [Bug fix 2]

## Breaking Changes

**[If any]** This release includes breaking changes. Please see the migration guide (link TBD).

## Documentation

- New documentation page (link TBD)
- Updated documentation page (link TBD)

## Integrations

- Integration 1 (link TBD) — New
- Integration 2 (link TBD) — Updated

## Examples

- Example 1 (link TBD) — New
- Example 2 (link TBD) — Updated

## Upgrading

From v0.YY.Z:
```bash
# Go modules
go get github.com/fall-out-bug/sdp@v0.XX.X

# Binary releases
curl -sSL https://github.com/fall-out-bug/sdp/releases/download/v0.XX.X/sdp-evidence_0.XX.X_linux_amd64.tar.gz | tar xz -C /usr/local/bin
```

## Known Issues

- [Known issue 1]
- [Known issue 2]

## Contributors

- @username1 (profile link TBD) — Contribution
- @username2 (profile link TBD) — Contribution

## Next Release

v0.ZZ.Z is planned for YYYY-MM-DD with [features].

## Demo

[YouTube video link]

## Changelog

Full changelog: [link to CHANGELOG.md]
```

---

## Appendix D: Quick Reference

### Key Commands

```bash
# Install CLI
go install github.com/fall-out-bug/sdp/cmd/sdp-evidence@latest

# Validate evidence
sdp-evidence validate --evidence .sdp/evidence/run-123.json

# Inspect evidence
sdp-evidence inspect --evidence .sdp/evidence/run-123.json

# Sign evidence (Sigstore)
sdp-sign sign --evidence .sdp/evidence/run-123.json

# Export evidence (SIEM)
sdp-export --format splunk --evidence .sdp/evidence/*.json
```

### Key URLs

| Resource | URL |
|----------|-----|
| GitHub repo | https://github.com/fall-out-bug/sdp |
| Documentation | https://docs.sdp.dev |
| Releases | https://github.com/fall-out-bug/sdp/releases |
| Issues | https://github.com/fall-out-bug/sdp/issues |
| Discussions | https://github.com/fall-out-bug/sdp/discussions |
| Discord | [Link] |
| Twitter | [@sdp_dev](https://twitter.com/sdp_dev) |

### Key People

| Role | GitHub | Twitter |
|------|--------|---------|
| Maintainer | @fall-out-bug | - |

### Key Dates

| Milestone | Week | Date |
|-----------|------|------|
| v0.10.0 CLI Release | 1 | YYYY-MM-DD |
| OpenCode Integration | 3 | YYYY-MM-DD |
| v0.11.0 Release | 4 | YYYY-MM-DD |
| in-toto Migration | 5 | YYYY-MM-DD |
| v0.12.0 Release | 6 | YYYY-MM-DD |
| v1.0.0-rc.1 Release | 8 | YYYY-MM-DD |
| v1.0.0 GA Release | 11 | YYYY-MM-DD |

---

## Final Notes

### Execution Discipline

This playbook is **not a suggestion** — it's an **execution contract**. Every week must ship:

1. **Release** (biweekly) — GitHub release with binaries, changelog, video
2. **Blog post** (weekly) — Technical tutorial or integration guide
3. **Social post** (daily, Mon-Fri) — Twitter/X, LinkedIn updates
4. **Community engagement** (weekly) — GitHub Discussions, Discord, Reddit

If any week does not ship, **pause and reassess**. The 12-week timeline assumes consistent execution.

### Adaptability

This playbook is **data-driven**. Track metrics weekly. If metrics are off by 20% or more, **adjust the plan**:

- **Stars off by 20%**: Increase Hacker News and Reddit frequency (weekly instead of biweekly)
- **Downloads off by 20%**: Improve documentation and examples (add more Getting Started guides)
- **Issues off by 20%**: Create GitHub Discussions templates and encourage community questions
- **Forks off by 20%**: Submit to more awesome lists and engage in ecosystem projects

### Community Feedback

This playbook is **community-informed**. Listen to feedback and adjust:

- **Feature requests**: Prioritize in roadmap if requested by 5+ users
- **Bug reports**: Fix critical bugs within 24 hours, non-critical within 1 week
- **Documentation gaps**: Add documentation within 3 days of request
- **Integration requests**: Build if requested by 3+ users or key ecosystem partner

### Success Definition

**Success is not 500 stars** — success is a **production-ready evidence layer** that developers actually use:

- 5+ CI integrations (GitHub Actions, GitLab CI, CircleCI, Jenkins, Azure Pipelines)
- 3+ ecosystem PRs merged (OpenCode, Beads, OhMyOpenCode)
- 20+ active GitHub Issues (community engagement)
- 200+ monthly CLI downloads (real usage)
- 90% documentation coverage (accessibility)

Stars are a leading indicator, but **usage is the true measure of success**.

---

**Playbook Version:** 1.0.0  
**Last Updated:** 2026-03-03  
**Author:** Research-backed execution plan based on successful OSS tools (OPA, Kyverno, k9s, in-toto, GUAC, GitHub CLI)  
**Execution Start:** Week 1 of playbook implementation
