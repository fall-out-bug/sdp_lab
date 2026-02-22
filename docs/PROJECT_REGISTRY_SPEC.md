# Project Registry Specification

Status: reference  
Source: `specs/project-registry.yaml`

## Overview

The project registry lists projects for the SDP swarm. Each project has a repo, workstreams, and model policy. Used by federation and orchestrator for multi-project scheduling.

## Project Fields

| Field | Type | Description |
|-------|------|-------------|
| id | string | Project identifier |
| repo_url | string | Git repo URL or `.` for local |
| repo_branch | string | Branch to use |
| beads_prefix | string | Beads issue prefix |
| language | string | Primary language (e.g. `go`) |
| workstreams | []string | Workstream labels |
| model_policy | string | Model selection policy (optional) |
| config | object | Extra config |

## Example

```yaml
projects:
  - id: sdp_dev
    repo_url: .
    repo_branch: main
    beads_prefix: sdp_dev
    language: go
    workstreams:
      - workstream:generic
      - workstream:builder
    model_policy: ""
    config: {}
```
