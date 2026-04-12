# Template / Generator Discipline — Next Step

Status: working next step
Date: 2026-03-23
Phase: 3

## Goal

Take the first thin implementation slice for template/generator discipline after stage-pack v1.

## Do now

1. Create a template inventory/classification doc:
   - identify canonical templates
   - identify generated/operator outputs
   - identify user-facing renderings
   - flag ambiguous assets

2. Add an explicit canonical launch-brief template:
   - `docs/templates/control-tower-launch-brief.template.md`

3. Wire the template discipline into the docs canon:
   - `docs/SESSION_START_CANON.md`
   - `packs/README.md`
   - any one or two related docs where this improves discoverability

4. Keep stage packs honest:
   - reference real templates only
   - avoid fake per-pack template systems unless such files actually exist

## Constraints

- no large generator framework
- no mass repo migration
- no fake automation surface
- no architecture rewrite
- keep it doc-first and narrow

## Desired outcome

After this slice, a new implementation session should be able to answer:
- which files are canonical source templates?
- what counts as generated operator output?
- what is only a rendering surface?
- what template should be used for a control-tower launch brief?
