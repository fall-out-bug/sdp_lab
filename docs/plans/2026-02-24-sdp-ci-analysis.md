# SDP CI Analysis: Duplication and Unnecessary Elements

**Date:** 2026-02-24  
**Scope:** sdp repo (submodule) + sdp_dev (sdp_lab) CI workflows

---

## 1. Task Breakdown

| Aspect | Description |
|--------|--------------|
| **Workflow inventory** | What runs where, on which triggers |
| **Duplication** | Same steps/jobs across workflows or repos |
| **Redundancy** | Steps that add little value or overlap |
| **Gaps** | Missing checks (e.g. unit tests not run) |
| **Release flow** | Two repos with release workflows — overlap? |

---

## 2. Workflow Inventory

### sdp repo (submodule, OSS)

| Workflow | Trigger | Jobs |
|----------|---------|------|
| `protocol-e2e.yml` | PR → main/dev (path-filtered) | protocol-e2e (Docker) |
| `go-release.yml` | Tag push `v*.*.*` | protocol-e2e (Docker) → release (GoReleaser, GPG, SLSA, SBOM) |

### sdp_dev (sdp_lab, private)

| Workflow | Trigger | Jobs |
|----------|---------|------|
| `ci.yml` | Push/PR → master, main, feature/* | build-test, evidence-gate, scope-gate, policy-gate, auto-attestation |
| `release.yml` | Tag push `v*` | protocol-e2e (Docker) → release (GoReleaser, cosign) |
| `skill-eval.yml` | PR (path-filtered: skills) | eval |

---

## 3. Findings

### 3.1 Duplication

#### A. Protocol E2E step — 3 places

The same conceptual step (Docker build + run protocol-e2e) appears in:

1. **sdp/protocol-e2e.yml** — PR gate  
2. **sdp/go-release.yml** — pre-release gate  
3. **sdp_dev/release.yml** — pre-release gate  

**Recommendation:** Extract to a reusable workflow, e.g. `.github/workflows/protocol-e2e-reusable.yml`:

```yaml
# protocol-e2e-reusable.yml
on:
  workflow_call:
jobs:
  protocol-e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          submodules: recursive  # for sdp_dev
      - name: Protocol E2E (Docker)
        env:
          GLM_API_KEY: ${{ secrets.GLM_API_KEY }}
        run: |
          docker build -f ci/Dockerfile.protocol-e2e -t sdp-protocol-e2e:latest .
          docker run --rm -e GLM_API_KEY="${GLM_API_KEY}" sdp-protocol-e2e:latest
```

**Caveat:** `ci/` path differs between sdp and sdp_dev (sdp uses `sdp/ci/`, sdp_dev uses root `ci/`). Reusable workflow would need to be called with correct `working-directory` or the caller would need to ensure `ci/` exists at build context root.

#### B. ci/ directory — near-duplicate in two repos

| Location | Used by | Difference |
|----------|---------|------------|
| `sdp/ci/` | sdp workflows | Dockerfile: `cd sdp-plugin`; test skips Phase 5 if no GLM_API_KEY |
| `sdp_dev/ci/` | sdp_dev release.yml | Dockerfile: `cd sdp/sdp-plugin`; test **fails** Phase 5 if no GLM_API_KEY |

- `protocol-e2e-test.sh` — almost identical (one behavioral difference in Phase 5).
- `Dockerfile.protocol-e2e` — same except sdp-plugin path.
- Fixtures — duplicated.

**Recommendation:** Treat `sdp/ci/` as source of truth. sdp_dev release could either:
- Use sdp as submodule and run from sdp root (if layout allows), or
- Keep a thin wrapper that delegates to sdp/ci (e.g. symlink or minimal copy).

---

### 3.2 Potentially Unnecessary

#### A. sdp_dev release.yml vs sdp go-release.yml

- **sdp** releases on `v*.*.*` (strict semver), uses GPG + SLSA + SBOM.
- **sdp_dev** releases on `v*`, uses cosign, no GPG.

If sdp is the canonical OSS release and sdp_dev only consumes it, `sdp_dev/release.yml` may be redundant. If sdp_dev publishes its own artifacts (sdp-evidence, sdp-guard, etc.), both are needed but could be clarified (e.g. "sdp CLI" vs "sdp_dev toolchain").

#### B. SLSA provenance + SBOM in go-release.yml

- `actions/attest-build-provenance@v1` — subject is `sdp-plugin/dist/*`.
- `anchore/sbom-action` — generates SBOM for sdp-plugin.

These support supply-chain security. Keep unless there is evidence they are unused. **Verdict:** likely useful, not unnecessary.

#### C. policy-gate in sdp_dev ci.yml

Uses hardcoded `/tmp/policy-input.json` with `evidence_validation_passed: true`, `scope_violations_count: 0`, etc. — not derived from actual evidence-gate/scope-gate results. The gate may not reflect real state.

**Recommendation:** Either wire real outputs from evidence-gate/scope-gate into policy input, or simplify/remove if the policy adds no value.

---

### 3.3 Gaps

#### A. sdp: no `go build` / `go test` for sdp-plugin on PR

`sdp` CI only runs protocol-e2e (Docker). Inside Docker, `protocol-e2e-test.sh` does not run `go test ./...` for sdp-plugin. Unit tests (e.g. `parse_integration_test`, `checker_test`) are never executed in sdp CI.

**Recommendation:** Add a job to `protocol-e2e.yml` (or a separate workflow) that runs:

```yaml
- name: Build and test sdp-plugin
  run: |
    cd sdp-plugin
    go build ./...
    go test ./... -count=1
```

#### B. Tag format mismatch

- sdp: `v*.*.*` (e.g. v0.1.0)
- sdp_dev: `v*` (e.g. v0.1)

If both are used for releases, consider aligning to semver (`v*.*.*`) for consistency.

---

## 4. Summary Table

| Issue | Severity | Action |
|-------|----------|--------|
| Protocol E2E duplicated 3× | Medium | Extract reusable workflow |
| ci/ duplicated sdp vs sdp_dev | Medium | Consolidate; sdp as source of truth |
| sdp: no go test for sdp-plugin | High | Add build+test job |
| policy-gate uses fake input | Low | Fix or remove |
| Two release flows (sdp vs sdp_dev) | Low | Document intent; align tag format |
| SLSA/SBOM | — | Keep |

---

## 5. Recommended Next Steps

1. **Immediate:** Add `go build` + `go test` for sdp-plugin to sdp CI (e.g. in `protocol-e2e.yml` or a new `build-test.yml`).
2. **Short-term:** Extract protocol-e2e to a reusable workflow; reduce copy-paste.
3. **Medium-term:** Consolidate `ci/` — sdp as canonical, sdp_dev references or wraps it.
4. **Optional:** Fix policy-gate to use real evidence/scope outputs, or simplify.

---

*Generated by @think analysis.*
