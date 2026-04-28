# SDP-Eval-Core v1 Runtime Contract

## Context Objects
- **Case**: Primary input object defining a single evaluation case (YAML)
- **Result**: Primary output object containing case execution outcome
- **Transcripts**: JSONL format with role+content per line

## Environment
- No required environment variables
- File system access required for transcript and case file I/O
- Case files located in `evals/cases/` or configurable directory

## Configuration
- Case definition format: YAML with fields:
  - `name`: case identifier
  - `skill`: associated skill name
  - `input_transcript`: path to JSONL transcript
  - `forbidden_patterns`: patterns that must NOT appear in output
  - `required_patterns`: patterns that MUST appear in output
  - `verdict`: expected outcome ("PASS" or "FAIL")

## Dependencies
- External: `gopkg.in/yaml.v3` for YAML case parsing
- Stdlib: `encoding/json` for JSONL transcript parsing

## Transcript Format
- JSONL (one JSON object per line)
- Each line: `{"role": "...", "content": "..."}` or nested message structure
- Roles: `assistant` messages are extracted for evaluation

## Evidence-Mismatch Metric (v1)
- Replaces hallucination rate per IIP council decision
- Measures governance-decision accuracy
- Computed as: (mismatched decisions) / (total decisions)
- Lower rates indicate better alignment between decisions and evidence
