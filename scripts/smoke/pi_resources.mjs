#!/usr/bin/env node
import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '..', '..');

function piModulePath() {
  if (process.env.PI_CODING_AGENT_MODULE) {
    return resolve(process.env.PI_CODING_AGENT_MODULE);
  }
  const npmRoot = execFileSync('npm', ['root', '-g'], { encoding: 'utf8' }).trim();
  return join(npmRoot, '@mariozechner', 'pi-coding-agent', 'dist', 'index.js');
}

function extractManifestNames(manifest, section) {
  const lines = manifest.split(/\r?\n/);
  const start = lines.findIndex((line) => line.trim() === `${section}:`);
  if (start === -1) return [];
  const names = [];
  for (let i = start + 1; i < lines.length; i++) {
    const line = lines[i];
    if (/^[a-zA-Z_][a-zA-Z0-9_]*:\s*$/.test(line)) break;
    const inline = line.match(/^\s*-\s*\{\s*name:\s*([^,}]+)/);
    if (inline) {
      names.push(inline[1].replace(/^['"]|['"]$/g, '').trim());
      continue;
    }
    const block = line.match(/^\s*name:\s*([^#]+)/);
    if (block) names.push(block[1].replace(/^['"]|['"]$/g, '').trim());
  }
  return [...new Set(names)].sort();
}

function missing(expected, actual) {
  const have = new Set(actual);
  return expected.filter((name) => !have.has(name));
}

const manifest = readFileSync(join(repoRoot, 'sdp.manifest.yaml'), 'utf8');
const expectedSkills = extractManifestNames(manifest, 'skills');
const expectedCommands = extractManifestNames(manifest, 'commands');

const pi = await import(pathToFileURL(piModulePath()).href);
const agentDir = mkdtempSync(join(tmpdir(), 'sdp-pi-agent-'));
try {
  const loader = new pi.DefaultResourceLoader({
    cwd: repoRoot,
    agentDir,
    noExtensions: true,
    noThemes: true,
    noContextFiles: true,
  });
  await loader.reload();

  const skillResult = loader.getSkills();
  const promptResult = loader.getPrompts();
  const actualSkills = skillResult.skills.map((skill) => skill.name).sort();
  const actualCommands = promptResult.prompts.map((prompt) => prompt.name).sort();
  const missingSkills = missing(expectedSkills, actualSkills);
  const missingCommands = missing(expectedCommands, actualCommands);
  const legacyPromptRefs = [];
  const promptsMissingArguments = [];
  for (const command of expectedCommands) {
    const body = readFileSync(join(repoRoot, '.pi', 'prompts', `${command}.md`), 'utf8');
    if (body.includes('.claude/skills/')) legacyPromptRefs.push(command);
    if (!body.includes('$ARGUMENTS')) promptsMissingArguments.push(command);
  }

  console.log(`pi skills: expected=${expectedSkills.length} actual=${actualSkills.length}`);
  console.log(`pi commands: expected=${expectedCommands.length} actual=${actualCommands.length}`);

  let failed = false;
  if (skillResult.diagnostics.length > 0) {
    failed = true;
    console.error('skill diagnostics:');
    for (const diagnostic of skillResult.diagnostics) console.error(JSON.stringify(diagnostic));
  }
  if (promptResult.diagnostics.length > 0) {
    failed = true;
    console.error('prompt diagnostics:');
    for (const diagnostic of promptResult.diagnostics) console.error(JSON.stringify(diagnostic));
  }
  if (missingSkills.length > 0) {
    failed = true;
    console.error(`missing skills: ${missingSkills.join(', ')}`);
  }
  if (missingCommands.length > 0) {
    failed = true;
    console.error(`missing commands: ${missingCommands.join(', ')}`);
  }
  if (legacyPromptRefs.length > 0) {
    failed = true;
    console.error(`pi prompts contain legacy .claude skill refs: ${legacyPromptRefs.join(', ')}`);
  }
  if (promptsMissingArguments.length > 0) {
    failed = true;
    console.error(`pi prompts missing $ARGUMENTS: ${promptsMissingArguments.join(', ')}`);
  }

  if (failed) process.exit(1);
  console.log('ok: Pi project resources match sdp.manifest.yaml');
} finally {
  rmSync(agentDir, { recursive: true, force: true });
}
