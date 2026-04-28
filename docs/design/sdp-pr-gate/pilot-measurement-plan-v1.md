# Pilot Measurement Plan v1

**Feature:** F151-06 (sdplab-hfk0.6)
**Internal namespace:** sdp-pr-gate
**Display name:** ChangePassport
**Status:** Design v1

## Overview

This document defines how the ChangePassport pilot will be measured. Every metric has a formula, a baseline window, a pilot window, a minimum sample size, and a stop/go rule.

Source: ChangePassport manifesto v2 §"Claims And Metrics" and SDP product layering memo v3 §"Discernment".

## Metrics

### 1. Install Time

| Attribute | Value |
|---|---|
| **Target** | ≤ 30 minutes |
| **Formula** | `install_time = time(first successful passport generated) - time(app installation completed)` |
| **Measurement** | Automated: app records `installation.completed_at` and `first_passport.generated_at` |
| **Baseline window** | N/A (no prior baseline) |
| **Pilot window** | First 2 weeks of pilot |
| **Minimum sample** | 3 installations |
| **Stop rule** | If median install time > 60 minutes after 3 attempts, stop and fix onboarding |
| **Go rule** | If all 3+ installations complete in ≤ 30 minutes, metric is green |

### 2. Passport Generation Time

| Attribute | Value |
|---|---|
| **Target** | ≤ 60 seconds after required checks finish |
| **Formula** | `gen_time = time(passport.generated_at) - time(last_required_check.completed_at)` |
| **Measurement** | Automated: timestamps in passport and evidence events |
| **Baseline window** | N/A |
| **Pilot window** | Full pilot duration |
| **Minimum sample** | 20 passports |
| **Stop rule** | If P95 gen_time > 120 seconds sustained over 1 week |
| **Go rule** | If P95 ≤ 60 seconds over 20+ passports |

### 3. Useful Decision Rate

| Attribute | Value |
|---|---|
| **Target** | ≥ 70% of passports drive reviewer action without manual reconstruction |
| **Formula** | `useful_rate = count(passports where reviewer acted based on passport) / count(total passports with reviewer response)` |
| **Measurement** | Post-PR survey (1 question: "Did the ChangePassport passport help you decide on this PR without reconstructing context from scratch?") + system signal (check viewed before decision) |
| **Baseline window** | 2 weeks before pilot (retrospective survey on current review process) |
| **Pilot window** | Weeks 3-4 of pilot |
| **Minimum sample** | 15 reviewed PRs with survey responses |
| **Stop rule** | If useful_rate < 40% after 15 responses |
| **Go rule** | If useful_rate ≥ 70% with 15+ responses |

### 4. Evidence-Mismatch Rate

| Attribute | Value |
|---|---|
| **Target** | < 5% |
| **Formula** | `mismatch_rate = count(passports with ≥1 evidence claim contradicting ground truth) / count(total passports)` |
| **Measurement** | Automated + spot-check. System: drift_detected flag. Human: weekly audit of 5 random passports, verify evidence claims against source systems. |
| **Baseline window** | N/A |
| **Pilot window** | Full pilot duration |
| **Minimum sample** | 50 passports (automated) + 10 spot-checks (human) |
| **Stop rule** | If mismatch_rate > 10% (systematic mismatch = critical failure) |
| **Go rule** | If mismatch_rate < 5% over 50+ passports |

**Evidence-mismatch vs hallucination**: The relevant failure mode is the passport asserting evidence that does not exist or contradicts ground truth. This is NOT an "hallucination rate" — the passport is not a content generator. It is an evidence reviewer.

### 5. False-Block Rate

| Attribute | Value |
|---|---|
| **Target** | < 5% |
| **Formula** | `false_block_rate = count(PRs blocked where reviewer decides the PR should have merged) / count(total PRs where system recommended hold/rework)` |
| **Measurement** | Override rate as proxy. Every override where original_decision was hold/rework and override reason indicates "should have passed" is a false block. |
| **Baseline window** | N/A |
| **Pilot window** | Full pilot duration |
| **Minimum sample** | 20 system-hold/rework decisions |
| **Stop rule** | If false_block_rate > 15% (system is blocking too many good PRs) |
| **Go rule** | If false_block_rate < 5% over 20+ system decisions |

### 6. Reviewer Time Delta

| Attribute | Value |
|---|---|
| **Target** | Median −20% in 4-week pilot |
| **Formula** | `time_delta = median(review_time_during_pilot) - median(review_time_baseline)`. Review time = time from PR assigned for review to decision (approve/request-changes/merge). |
| **Measurement** | GitHub review timestamps (automated). Exclude PRs with < 5 minutes review time (rubber stamps). Exclude PRs with > 4 hours review time (unrelated delays). |
| **Baseline window** | 4 weeks before pilot start |
| **Pilot window** | 4 weeks of pilot |
| **Minimum sample** | 30 reviewed PRs in baseline, 30 in pilot |
| **Stop rule** | If reviewer time increases by > 10% (passport adds overhead without value) |
| **Go rule** | If median review time decreased by ≥ 20% |

### 7. Post-Merge Incident Rate

| Attribute | Value |
|---|---|
| **Target** | Not above baseline |
| **Formula** | `incident_rate = count(post-merge incidents within 7 days) / count(merged PRs)`. Incident = rollback, hotfix, or severity-1/2 bug traced to a merged PR. |
| **Measurement** | Incident tracking system. Each incident is attributed to the PR that introduced it. |
| **Baseline window** | 4 weeks before pilot start |
| **Pilot window** | 4 weeks of pilot + 1 week follow-up |
| **Minimum sample** | 50 merged PRs in each window |
| **Stop rule** | If incident_rate increases by > 50% over baseline |
| **Go rule** | If incident_rate ≤ baseline rate |

## Overall Pilot Structure

### Duration

4 weeks minimum. Extend to 6 weeks if any metric has < minimum sample size at week 4.

### Minimum participants

1 pilot customer (team of 10-50 engineers) with ≥3 active repositories.

### Data collection method

| Data | Method | Frequency |
|---|---|---|
| Install time | Automated (app) | Per installation |
| Passport gen time | Automated (app) | Per passport |
| Useful decision rate | Survey (1-question) + system | Per reviewed PR |
| Evidence-mismatch rate | Automated drift + weekly audit | Continuous + weekly |
| False-block rate | Override analysis | Continuous |
| Reviewer time | GitHub timestamps | Per PR |
| Post-merge incidents | Incident tracker | Weekly |

### Privacy considerations

- No PR content is stored (only metadata and evidence references)
- Survey responses are anonymized
- Reviewer time is aggregated (no per-reviewer tracking)
- All data is stored in the pilot customer's infrastructure
- Data is deleted 30 days after pilot completion unless customer opts in to retention

## Stop/Go Rules

### Stop the pilot (abort)

ANY of:
- Evidence-mismatch rate > 10% (systematic failure)
- Post-merge incident rate increases > 50% (product introduces risk)
- Reviewer time increases > 10% (product adds overhead)
- Pilot customer requests termination

### Extend the pilot

ANY of:
- Any metric has < minimum sample size at week 4
- Install time is improving but not yet at target
- Useful decision rate is between 50-70% (promising but not proven)

### Declare pilot success (proceed to GA evaluation)

ALL of:
- Install time ≤ 30 minutes (3+ installations)
- Passport gen time P95 ≤ 60 seconds (20+ passports)
- Useful decision rate ≥ 70% (15+ responses)
- Evidence-mismatch rate < 5% (50+ passports)
- False-block rate < 5% (20+ decisions)
- Reviewer time decreased by ≥ 20% (30+ PRs)
- Post-merge incident rate ≤ baseline (50+ merged PRs)

## GA Criteria

Pilot success is necessary but not sufficient for GA. Additional GA requirements:
- Schema v1 freeze complete and published
- Evidence Provider API v1 published with ≥2 provider integrations
- Decision Record v1 published
- Override protocol documented and tested
- Security review completed
- Documentation complete (install guide, provider integration guide, override guide)
- Pricing hypothesis validated (WTP data from pilot)

GA decision is made by the product owner based on pilot data + commercial signal.
