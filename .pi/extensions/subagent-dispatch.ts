import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type } from "typebox";
import { execFile } from "node:child_process";
import { existsSync } from "node:fs";
import { dirname, join } from "node:path";

// ── Helpers ────────────────────────────────────────────────────────────────

function findProjectRoot(cwd: string): string {
  let dir = cwd;
  while (dir !== dirname(dir)) {
    if (existsSync(join(dir, "sdp.manifest.yaml")) || existsSync(join(dir, "go.mod"))) {
      return dir;
    }
    dir = dirname(dir);
  }
  return cwd;
}

function execTool(
  cmd: string,
  args: string[],
  cwd: string,
  timeoutMs = 120000
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve) => {
    execFile(cmd, args, { cwd, timeout: timeoutMs }, (error, stdout, stderr) => {
      resolve({
        stdout: stdout?.toString() ?? "",
        stderr: stderr?.toString() ?? "",
        exitCode: error ? ((error as any).code ? parseInt((error as any).code, 10) || 1 : 1) : 0,
      });
    });
  });
}

function findHarnessBinary(root: string): { cmd: string; isGoRun: boolean } {
  const localBin = join(root, "bin", "sdp-harness");
  if (existsSync(localBin)) return { cmd: localBin, isGoRun: false };
  const localRoot = join(root, "sdp-harness");
  if (existsSync(localRoot)) return { cmd: localRoot, isGoRun: false };
  return { cmd: "go", isGoRun: true };
}

// ── Extension ──────────────────────────────────────────────────────────────

export default function (pi: ExtensionAPI) {

  // ── Tool: sdp_subagent ───────────────────────────────────────────────────
  pi.registerTool({
    name: "sdp_subagent",
    label: "SDP Subagent",
    description:
      "Dispatch a bounded subagent session via sdp-harness. " +
      "Creates a session for a specific workstream leaf, runs one phase turn, " +
      "and returns the result. Use for parallel or isolated execution.",
    parameters: Type.Object({
      feature: Type.String({
        description: "Feature ID (e.g. F150)",
      }),
      ws: Type.String({
        description: "Workstream ID (leaf workstream to execute)",
      }),
      prompt: Type.String({
        description: "The task/prompt to execute in the subagent session",
      }),
      sessionPrefix: Type.Optional(Type.String({
        description: "Session ID prefix (default: 'subagent')",
        default: "subagent",
      })),
      releaseAfter: Type.Optional(Type.Boolean({
        description: "Release the claim after execution (default: true)",
        default: true,
      })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const harness = findHarnessBinary(root);
      const sessionId = `${params.sessionPrefix}-${Date.now()}`;

      // Step 1: Compile lock if needed
      onUpdate?.({ toolCallId, content: "Checking workgraph lock..." });
      const lockPath = join(root, ".sdp", "workgraph.lock.json");
      if (!existsSync(lockPath)) {
        const compileArgs = harness.isGoRun
          ? ["run", "./cmd/sdp-harness", "compile-lock", "--project-root", root]
          : ["compile-lock", "--project-root", root];
        await execTool(harness.cmd, compileArgs, root, 30000);
      }

      // Step 2: Create session
      onUpdate?.({ toolCallId, content: `Creating session ${sessionId} for ${params.feature}/${params.ws}...` });
      const newArgs = harness.isGoRun
        ? ["run", "./cmd/sdp-harness", "new",
            "--session", sessionId,
            "--project-root", root,
            "--feature", params.feature,
            "--ws", params.ws]
        : ["new",
            "--session", sessionId,
            "--project-root", root,
            "--feature", params.feature,
            "--ws", params.ws];

      const newResult = await execTool(harness.cmd, newArgs, root, 30000);
      if (newResult.exitCode !== 0) {
        return {
          content: [{ type: "text", text: `Failed to create session:\n${newResult.stderr || newResult.stdout}` }],
          details: { exitCode: newResult.exitCode, phase: "new" },
        };
      }

      // Step 3: Run phase
      onUpdate?.({ toolCallId, content: `Running phase turn...` });
      const runArgs = harness.isGoRun
        ? ["run", "./cmd/sdp-harness", "run",
            "--session", sessionId,
            "--prompt", params.prompt]
        : ["run",
            "--session", sessionId,
            "--prompt", params.prompt];

      const runResult = await execTool(harness.cmd, runArgs, root, 300000);

      // Step 4: Get events
      onUpdate?.({ toolCallId, content: `Fetching session events...` });
      const eventsArgs = harness.isGoRun
        ? ["run", "./cmd/sdp-harness", "events", "--session", sessionId]
        : ["events", "--session", sessionId];
      const eventsResult = await execTool(harness.cmd, eventsArgs, root, 15000);

      // Step 5: Release if requested
      if (params.releaseAfter !== false) {
        onUpdate?.({ toolCallId, content: `Releasing claim...` });
        const releaseArgs = harness.isGoRun
          ? ["run", "./cmd/sdp-harness", "release", "--session", sessionId]
          : ["release", "--session", sessionId];
        await execTool(harness.cmd, releaseArgs, root, 15000);
      }

      const output = [
        "## Subagent Execution Complete",
        `Session: ${sessionId}`,
        `Feature: ${params.feature}`,
        `Workstream: ${params.ws}`,
        "",
        "### Phase Output",
        runResult.stdout || runResult.stderr,
        "",
        "### Session Events",
        eventsResult.stdout || eventsResult.stderr,
      ].join("\n");

      return {
        content: [{ type: "text", text: output }],
        details: {
          exitCode: runResult.exitCode,
          sessionId,
          feature: params.feature,
          ws: params.ws,
          released: params.releaseAfter !== false,
        },
      };
    },
  });

  // ── Command: /subagent ───────────────────────────────────────────────────
  pi.registerCommand("subagent", {
    description: "Dispatch a bounded subagent: /subagent --feature=F150 --ws=WS-01 \"implement auth\"",
    handler: async (args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const harness = findHarnessBinary(root);

      // Parse: --feature=X --ws=Y "prompt text"
      const featureMatch = args.match(/--feature=(\S+)/);
      const wsMatch = args.match(/--ws=(\S+)/);
      const promptMatch = args.match(/--ws=\S+\s+(.+)/);

      if (!featureMatch || !wsMatch || !promptMatch) {
        ctx.ui.notify("Usage: /subagent --feature=F150 --ws=WS-01 \"your prompt\"", "error");
        return;
      }

      const feature = featureMatch[1];
      const ws = wsMatch[1];
      const prompt = promptMatch[1];
      const sessionId = `pi-subagent-${Date.now()}`;

      ctx.ui.notify(`Creating subagent session ${sessionId}...`, "info");

      // Create
      const newArgs = harness.isGoRun
        ? ["run", "./cmd/sdp-harness", "new", "--session", sessionId, "--project-root", root, "--feature", feature, "--ws", ws]
        : ["new", "--session", sessionId, "--project-root", root, "--feature", feature, "--ws", ws];
      const newResult = await execTool(harness.cmd, newArgs, root, 30000);
      if (newResult.exitCode !== 0) {
        ctx.ui.notify(`Failed to create session: ${newResult.stderr || newResult.stdout}`, "error");
        return;
      }

      ctx.ui.notify(`Running phase for ${feature}/${ws}...`, "info");

      // Run
      const runArgs = harness.isGoRun
        ? ["run", "./cmd/sdp-harness", "run", "--session", sessionId, "--prompt", prompt]
        : ["run", "--session", sessionId, "--prompt", prompt];
      const runResult = await execTool(harness.cmd, runArgs, root, 300000);

      ctx.ui.notify(
        runResult.exitCode === 0 ? "Subagent complete ✓" : "Subagent finished with issues",
        runResult.exitCode === 0 ? "success" : "warning"
      );

      // Release
      const releaseArgs = harness.isGoRun
        ? ["run", "./cmd/sdp-harness", "release", "--session", sessionId]
        : ["release", "--session", sessionId];
      await execTool(harness.cmd, releaseArgs, root, 15000);

      ctx.ui.setEditorText(runResult.stdout || runResult.stderr);
    },
  });

  // ── Command: /parallel ───────────────────────────────────────────────────
  pi.registerCommand("parallel", {
    description: "Run multiple subagents in parallel for different workstreams",
    handler: async (args, ctx) => {
      // Parse: --feature=F150 --ws=WS-01,WS-02,WS-03 "prompt"
      const featureMatch = args.match(/--feature=(\S+)/);
      const wsMatch = args.match(/--ws=([\S,]+)/);
      const promptMatch = args.match(/--ws=[\S,]+\s+(.+)/);

      if (!featureMatch || !wsMatch || !promptMatch) {
        ctx.ui.notify("Usage: /parallel --feature=F150 --ws=WS-01,WS-02,WS-03 \"your prompt\"", "error");
        return;
      }

      const feature = featureMatch[1];
      const wsList = wsMatch[1].split(",");
      const prompt = promptMatch[1];
      const root = findProjectRoot(ctx.cwd);
      const harness = findHarnessBinary(root);

      ctx.ui.notify(`Dispatching ${wsList.length} parallel subagents...`, "info");

      const results = await Promise.all(
        wsList.map(async (ws) => {
          const sessionId = `pi-par-${ws}-${Date.now()}`;
          // Create
          const newArgs = harness.isGoRun
            ? ["run", "./cmd/sdp-harness", "new", "--session", sessionId, "--project-root", root, "--feature", feature, "--ws", ws.trim()]
            : ["new", "--session", sessionId, "--project-root", root, "--feature", feature, "--ws", ws.trim()];
          await execTool(harness.cmd, newArgs, root, 30000);

          // Run
          const runArgs = harness.isGoRun
            ? ["run", "./cmd/sdp-harness", "run", "--session", sessionId, "--prompt", prompt]
            : ["run", "--session", sessionId, "--prompt", prompt];
          const result = await execTool(harness.cmd, runArgs, root, 300000);

          // Release
          const relArgs = harness.isGoRun
            ? ["run", "./cmd/sdp-harness", "release", "--session", sessionId]
            : ["release", "--session", sessionId];
          await execTool(harness.cmd, relArgs, root, 15000);

          return { ws: ws.trim(), exitCode: result.exitCode, output: result.stdout || result.stderr };
        })
      );

      const summary = results
        .map((r) => `${r.ws}: ${r.exitCode === 0 ? "✓" : "✗"}`)
        .join(" | ");
      ctx.ui.notify(summary, "info");

      const allOutput = results
        .map((r) => `## ${r.ws}\n${r.output}`)
        .join("\n\n");
      ctx.ui.setEditorText(allOutput);
    },
  });
}
