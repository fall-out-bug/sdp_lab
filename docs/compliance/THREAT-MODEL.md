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
| Unauthorized scope expansion | scope-gate (fail-closed) + WS frontmatter validation | Admin can modify gate config |
| Evidence tampering | SHA-256 content hashes + in-toto attestation format | CI runner compromise can forge |
| Gate suppression | push-protection + required status checks | GitHub admin can disable branch protection |
| Configuration drift | `sdp doctor` periodic check + version-controlled guard-rules | Social engineering of operators |
| Credential leakage | secretscan gate (fail-closed) | Scanning is pattern-based, not semantic |

## Out of Scope

- Runtime security of deployed applications
- Network-level threats (MITM, DDoS)
- Supply chain attacks on Go dependencies (use `govulncheck` separately)
- Physical access attacks on developer machines

## Canonical Statement

SDP provides **tamper-evident** (not tamper-proof) evidence and **fail-closed** (not bypass-proof) gates. The security model assumes the CI environment and git hosting platform are trusted. See [trust-guarantees.md](../reference/trust-guarantees.md) for the full canonical wording.

---

*Source: [trust-guarantees.md](../reference/trust-guarantees.md), [ci-gates-map.md](../reference/ci-gates-map.md)*
