# SDP Threat Model

> **Last updated:** 2026-04-26
> **Scope:** Trust surface — evidence integrity, gate enforcement, scope containment

## Threat Actors

| Actor | Capability | Motivation |
|-------|-----------|------------|
| Malicious contributor | Write access to feature branches | Bypass scope checks, inject unauthorized changes |
| Compromised CI runner | Full CI environment access | Forge evidence artifacts, suppress gate failures |
| Insider with admin access | Repository admin privileges | Modify gate configuration, disable protection |
| External attacker | No repo access | Exploit published artifacts |

## Trust Boundaries

```
┌─────────────────────────────────────────────┐
│                SDP Trust Boundary           │
│                                             │
│  ┌─────────┐    ┌──────────┐               │
│  │  Gates  │───▶│ Evidence │               │
│  └────┬────┘    └────┬─────┘               │
│       │              │                      │
│  ┌────▼────┐    ┌────▼─────┐               │
│  │  Scope  │    │ Attest.  │               │
│  │  Guard  │    │ (in-toto)│               │
│  └─────────┘    └──────────┘               │
│                                             │
└──────────────────┬──────────────────────────┘
                   │
          ┌────────▼────────┐
          │   Git Repository  │
          │  (outside SDP)    │
          └───────────────────┘
```

## Mitigations

| Threat | Mitigation | Residual Risk |
|--------|-----------|---------------|
| Unauthorized scope expansion | scope-gate (`go run ./cmd/sdp-guard --ws`) + required-checks | Admin can modify CI workflow |
| Evidence tampering | SHA-256 content hashes + in-toto attestation + Sigstore keyless signing | CI runner compromise can forge before signing |
| Gate suppression | push-protection + required-checks validates all 12 gate jobs | GitHub admin can disable branch protection |
| Configuration drift | Version-controlled `.github/workflows/ci.yml` + `.sdp/policies/` | Social engineering of operators |
| Coverage regression | coverage-gate compares against baseline with -2pp threshold | Baseline file can be modified on main |

## Out of Scope

- Runtime security of deployed applications
- Network-level threats (MITM, DDoS)
- Supply chain attacks on Go dependencies (use `govulncheck` separately)
- Physical access attacks on developer machines
- Secret scanning (no dedicated CI gate; rely on GitHub native secret scanning or external tools)

## Canonical Statement

SDP provides **tamper-evident** (not tamper-proof) evidence and **configurable** (not always fail-closed) gates. The default policy enforcement mode is advisory; blocking mode requires explicit configuration. The security model assumes the CI environment and git hosting platform are trusted. See [trust-guarantees.md](../reference/trust-guarantees.md) for the full canonical wording.

---

*Source: [trust-guarantees.md](../reference/trust-guarantees.md), [ci-gates-map.md](../reference/ci-gates-map.md)*
