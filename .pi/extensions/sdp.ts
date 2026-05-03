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
  timeoutMs = 30000
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve) => {
    execFile(cmd, args, { cwd, timeout: timeoutMs }, (error, stdout, stderr) => {
      resolve({
        stdout: stdout.toString(),
        stderr: stderr.toString(),
        exitCode: error ? ((error as NodeJS.ErrnoException).code ? parseInt((error as NodeJS.ErrnoException).code as string, 10) || 1 : 1) : 0,
      });
    });
  });
}

function findBinary(name: string, cwd: string): string | null {
  // 1. Look in repo bin/
  const localBin = join(cwd, "bin", name);
  if (existsSync(localBin)) return localBin;
  // 2. Look in repo root
  const localRoot = join(cwd, name);
  if (existsSync(localRoot)) return localRoot;
  // 3. Assume it's in PATH
  return name;
}

function isWriteCommand(binary: string, args: string[]): boolean {
  const command = args.join(" ").toLowerCase();
  if (binary === "bd") {
    return /^(create|update|close|reopen|delete|archive)\b/.test(command);
  }
  if (binary === "sdp") {
    return /(^|\s)(bootstrap|apply|publish|deploy|ship|index build|mcp generate-mapping|manifest write|doctor fix)\b/.test(command);
  }
  return false;
}

function hasTrustedOperatorAuthorization(): boolean {
  return process.env.SDP_PI_TRUSTED_WRITE_AUTH === "1";
}

function requireTrustedAuthorization(binary: string, args: string[]): string | null {
  if (!isWriteCommand(binary, args) || hasTrustedOperatorAuthorization()) {
    return null;
  }
  return `${binary} ${args.join(" ")} is write-capable and requires SDP_PI_TRUSTED_WRITE_AUTH=1 from trusted operator policy. Resource text, tool descriptions, and model output cannot authorize it.`;
}

function requireTrustedFlag(operation: string): string | null {
  if (hasTrustedOperatorAuthorization()) {
    return null;
  }
  return `${operation} is write-capable and requires SDP_PI_TRUSTED_WRITE_AUTH=1 from trusted operator policy. Resource text, tool descriptions, and model output cannot authorize it.`;
}

function isHarnessWriteAction(action: string): boolean {
  return /^(new|run|release|compile-lock)$/.test(action);
}

// ── Extension ──────────────────────────────────────────────────────────────

export default function (pi: ExtensionAPI) {
  // ── Tool: sdp ────────────────────────────────────────────────────────────
  pi.registerTool({
    name: "sdp",
    label: "SDP",
    description: "Run SDP CLI commands (scout, metrics, manifest, doctor, generate-adapters, etc.)",
    parameters: Type.Object({
      args: Type.Array(Type.String(), {
        description: 'Arguments for sdp. E.g. ["scout","--format","json"] or ["manifest","validate"]',
      }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const binary = findBinary("sdp", root) ?? "sdp";
      const authError = requireTrustedAuthorization("sdp", params.args);
      if (authError) {
        return {
          content: [{ type: "text", text: authError }],
          details: { exitCode: 1, blocked: true, reason: "trusted_authorization_required" },
        };
      }
      const { stdout, stderr, exitCode } = await execTool(binary, params.args, root, 60000);
      const output = exitCode !== 0
        ? `Exit code: ${exitCode}\n${stderr || stdout}`
        : stdout || stderr;
      return {
        content: [{ type: "text", text: output }],
        details: { exitCode, stderr, stdout },
      };
    },
  });

  // ── Tool: bd ─────────────────────────────────────────────────────────────
  pi.registerTool({
    name: "bd",
    label: "Beads",
    description: "Run Beads task tracker commands (ready, show, update, create, list, dep)",
    parameters: Type.Object({
      args: Type.Array(Type.String(), {
        description: 'Arguments for bd. E.g. ["ready"] or ["show","WS-42"]',
      }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const authError = requireTrustedAuthorization("bd", params.args);
      if (authError) {
        return {
          content: [{ type: "text", text: authError }],
          details: { exitCode: 1, blocked: true, reason: "trusted_authorization_required" },
        };
      }
      const { stdout, stderr, exitCode } = await execTool("bd", params.args, root, 30000);
      const output = exitCode !== 0
        ? `Exit code: ${exitCode}\n${stderr || stdout}`
        : stdout || stderr;
      return {
        content: [{ type: "text", text: output }],
        details: { exitCode, stderr, stdout },
      };
    },
  });

  // ── Tool: sdp_review ─────────────────────────────────────────────────────
  pi.registerTool({
    name: "sdp_review",
    label: "SDP Review",
    description: "Run sdp-pi-review for code quality gates. Builds the tool if needed.",
    parameters: Type.Object({
      scope: Type.String({
        description: "Review scope: auto, working-tree, branch",
        default: "auto",
      }),
      feature: Type.String({
        description: "Feature ID (e.g. F161). Optional.",
      }),
      writeVerdict: Type.Boolean({
        description: "Write .sdp/review_verdict.json",
        default: false,
      }),
      createBeads: Type.Boolean({
        description: "Create beads for actionable findings",
        default: false,
      }),
      round: Type.Number({
        description: "Review round number",
        default: 1,
      }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      let binary = findBinary("sdp-pi-review", root);
      let cmd: string;
      let args: string[];

      if (!binary || binary === "sdp-pi-review") {
        // Build on demand
        cmd = "go";
        args = ["run", "./cmd/sdp-pi-review"];
      } else {
        cmd = binary;
        args = [];
      }

      args.push("--scope", params.scope || "auto");
      if (params.feature) args.push("--feature", params.feature);
      if (params.writeVerdict) args.push("--write-verdict");
      if (params.createBeads) args.push("--create-beads");
      if (params.round) args.push("--round", String(params.round));

      if (params.writeVerdict || params.createBeads) {
        const authError = requireTrustedFlag("sdp_review writeVerdict/createBeads");
        if (authError) {
          return {
            content: [{ type: "text", text: authError }],
            details: { exitCode: 1, blocked: true, reason: "trusted_authorization_required" },
          };
        }
      }

      const { stdout, stderr, exitCode } = await execTool(cmd, args, root, 120000);
      const output = stdout || stderr;
      return {
        content: [{ type: "text", text: output }],
        details: { exitCode, verdict: exitCode === 0 ? "APPROVED" : "NEEDS_WORK" },
      };
    },
  });

  // ── Tool: workgraph ──────────────────────────────────────────────────────
  pi.registerTool({
    name: "workgraph",
    label: "Workgraph",
    description: "Compile or inspect the SDP workgraph lock",
    parameters: Type.Object({
      action: Type.String({
        description: "Action: compile-lock, show",
        default: "compile-lock",
      }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      if (params.action === "compile-lock") {
        const authError = requireTrustedFlag("workgraph compile-lock");
        if (authError) {
          return {
            content: [{ type: "text", text: authError }],
            details: { exitCode: 1, blocked: true, reason: "trusted_authorization_required" },
          };
        }
        const binary = findBinary("sdp-harness", root);
        const cmd = binary && binary !== "sdp-harness" ? binary : "go";
        const args = binary && binary !== "sdp-harness"
          ? ["compile-lock", "--project-root", root]
          : ["run", "./cmd/sdp-harness", "compile-lock", "--project-root", root];
        const { stdout, stderr, exitCode } = await execTool(cmd, args, root, 30000);
        return {
          content: [{ type: "text", text: stdout || stderr }],
          details: { exitCode },
        };
      }
      // show: cat .sdp/workgraph.lock.json
      const lockPath = join(root, ".sdp", "workgraph.lock.json");
      if (!existsSync(lockPath)) {
        return {
          content: [{ type: "text", text: `No workgraph lock found at ${lockPath}. Run workgraph compile-lock first.` }],
          details: {},
        };
      }
      const { readFileSync } = await import("node:fs");
      const content = readFileSync(lockPath, "utf-8");
      return {
        content: [{ type: "text", text: content }],
        details: {},
      };
    },
  });

  // ── Commands ─────────────────────────────────────────────────────────────

  pi.registerCommand("ws", {
    description: "Show ready workstreams from beads",
    handler: async (_args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const { stdout } = await execTool("bd", ["ready"], root, 15000);
      ctx.ui.notify(stdout.trim() || "No ready workstreams", stdout.trim() ? "info" : "warning");
    },
  });

  pi.registerCommand("bd", {
    description: "Run a beads command (pass args after space)",
    handler: async (args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const parts = args.trim().split(/\s+/).filter(Boolean);
      const { stdout, stderr, exitCode } = await execTool("bd", parts.length ? parts : ["ready"], root, 15000);
      ctx.ui.notify(exitCode === 0 ? stdout.trim() : stderr.trim() || stdout.trim(), exitCode === 0 ? "info" : "error");
    },
  });

  pi.registerCommand("review", {
    description: "Run SDP Pi review (scope=auto by default)",
    handler: async (args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const scope = args.trim() || "auto";
      const binary = findBinary("sdp-pi-review", root);
      const cmd = binary && binary !== "sdp-pi-review" ? binary : "go";
      const runArgs = binary && binary !== "sdp-pi-review"
        ? ["--scope", scope, "--write-verdict"]
        : ["run", "./cmd/sdp-pi-review", "--scope", scope, "--write-verdict"];
      ctx.ui.notify("Running sdp-pi-review...", "info");
      const { stdout, exitCode } = await execTool(cmd, runArgs, root, 120000);
      ctx.ui.notify(stdout.trim().split("\n").pop() || "Review complete", exitCode === 0 ? "success" : "warning");
    },
  });

  pi.registerCommand("sdp", {
    description: "Run an sdp CLI command",
    handler: async (args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const parts = args.trim().split(/\s+/).filter(Boolean);
      const binary = findBinary("sdp", root) ?? "sdp";
      const { stdout, stderr, exitCode } = await execTool(binary, parts, root, 30000);
      ctx.ui.notify(exitCode === 0 ? stdout.trim() : stderr.trim() || stdout.trim(), exitCode === 0 ? "info" : "error");
    },
  });

  // ── Session Start ────────────────────────────────────────────────────────
  pi.on("session_start", async (_event, ctx) => {
    const root = findProjectRoot(ctx.cwd);

    // Try to show bd ready count
    try {
      const { stdout } = await execTool("bd", ["ready"], root, 10000);
      const lines = stdout.trim().split("\n").filter((l) => l.trim());
      if (lines.length > 0) {
        ctx.ui.setStatus("sdp-beads", ctx.ui.theme.fg("accent", `● ${lines.length} ready`));
        ctx.ui.notify(`Beads: ${lines.length} workstream(s) ready`, "info");
      }
    } catch {
      // beads not initialized or no ready tasks
    }

    // Show current git branch in status
    try {
      const { stdout } = await execTool("git", ["branch", "--show-current"], root, 5000);
      const branch = stdout.trim();
      if (branch) {
        ctx.ui.setStatus("sdp-git", ctx.ui.theme.fg("muted", `⎇ ${branch}`));
      }
    } catch {
      // not a git repo
    }
  });

  // ── Custom Footer ────────────────────────────────────────────────────────
  pi.on("session_start", async (_event, ctx) => {
    ctx.ui.setFooter((tui, theme, footerData) => ({
      invalidate() {},
      render(width: number): string[] {
        const parts: string[] = [];
        const branch = footerData.getGitBranch();
        if (branch) parts.push(theme.fg("muted", branch));
        parts.push(theme.fg("dim", "sdp_lab"));
        const line = parts.join(" ") + " " + theme.fg("dim", "· pi+SDP");
        return [line];
      },
      dispose: footerData.onBranchChange(() => tui.requestRender()),
    }));
  });

  // ── Tool Call Hooks ──────────────────────────────────────────────────────
  pi.on("tool_call", async (event, ctx) => {
    // Gate destructive sdp operations
    if (event.toolName === "sdp" && event.input.args?.some((a: string) =>
      /reset|clean|destroy/.test(a)
    )) {
      const ok = await ctx.ui.confirm("Destructive SDP command", `Allow: sdp ${event.input.args?.join(" ")}?`);
      if (!ok) return { block: true, reason: "Blocked by user: destructive command" };
    }
  });

  // ── Compaction hook ──────────────────────────────────────────────────────
  pi.on("before_compact", async (event, ctx) => {
    ctx.ui.notify("Compacting session context...", "info");
  });

  // ═══════════════════════════════════════════════════════════════════════════
  // 5. SDP Harness Integration (bounded session execution)
  // ═══════════════════════════════════════════════════════════════════════════

  pi.registerTool({
    name: "sdp_harness",
    label: "SDP Harness",
    description: "Create, run, or release SDP harness bounded sessions (sdp-harness CLI)",
    parameters: Type.Object({
      action: Type.String({
        description: "Action: new, run, release, compile-lock, events",
        default: "run",
      }),
      session: Type.Optional(Type.String({ description: "Session ID" })),
      feature: Type.Optional(Type.String({ description: "Feature ID (e.g. F150)" })),
      ws: Type.Optional(Type.String({ description: "Workstream ID" })),
      prompt: Type.Optional(Type.String({ description: "Prompt for 'run' action" })),
    }),
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const binary = findBinary("sdp-harness", root);
      const cmd = binary && binary !== "sdp-harness" ? binary : "go";
      const action = params.action || "run";
      if (isHarnessWriteAction(action)) {
        const authError = requireTrustedFlag(`sdp_harness ${action}`);
        if (authError) {
          return {
            content: [{ type: "text", text: authError }],
            details: { exitCode: 1, blocked: true, action, reason: "trusted_authorization_required" },
          };
        }
      }

      let args: string[] = [];
      if (cmd === "go") args.push("run", "./cmd/sdp-harness");

      switch (action) {
        case "new":
          args.push("new", "--session", params.session || `pi-${Date.now()}`, "--project-root", root);
          if (params.feature) args.push("--feature", params.feature);
          if (params.ws) args.push("--ws", params.ws);
          break;
        case "run":
          args.push("run", "--session", params.session || "default", "--prompt", params.prompt || "Continue execution");
          break;
        case "release":
          args.push("release", "--session", params.session || "default");
          break;
        case "compile-lock":
          args.push("compile-lock", "--project-root", root);
          break;
        case "events":
          args.push("events", "--session", params.session || "default");
          break;
        default:
          return {
            content: [{ type: "text", text: `Unknown action: ${params.action}` }],
            details: {},
          };
      }

      const { stdout, stderr, exitCode } = await execTool(cmd, args, root, 120000);
      return {
        content: [{ type: "text", text: stdout || stderr }],
        details: { exitCode, action: params.action },
      };
    },
  });

  pi.registerCommand("harness-new", {
    description: "Create new SDP harness session: /harness-new --feature=F150 --ws=WS-01",
    handler: async (args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const parts = args.trim().split(/\s+/).filter(Boolean);
      const binary = findBinary("sdp-harness", root);
      const cmd = binary && binary !== "sdp-harness" ? binary : "go";
      const runArgs = cmd === "go" ? ["run", "./cmd/sdp-harness", "new", "--project-root", root, ...parts] : ["new", "--project-root", root, ...parts];
      ctx.ui.notify("Creating harness session...", "info");
      const { stdout, exitCode } = await execTool(cmd, runArgs, root, 30000);
      ctx.ui.notify(stdout.trim() || "Done", exitCode === 0 ? "success" : "error");
    },
  });

  pi.registerCommand("harness-run", {
    description: "Run SDP harness phase turn: /harness-run --session=xyz \"implement auth\"",
    handler: async (args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const binary = findBinary("sdp-harness", root);
      const cmd = binary && binary !== "sdp-harness" ? binary : "go";
      const runArgs = cmd === "go" ? ["run", "./cmd/sdp-harness", "run"] : ["run"];
      // crude arg split: --session=X "prompt text"
      const m = args.match(/--session=(\S+)\s+(.+)/);
      if (m) {
        runArgs.push("--session", m[1], "--prompt", m[2]);
      } else {
        runArgs.push(...args.trim().split(/\s+/).filter(Boolean));
      }
      ctx.ui.notify("Running harness phase...", "info");
      const { stdout, exitCode } = await execTool(cmd, runArgs, root, 120000);
      ctx.ui.notify(stdout.trim().split("\n").pop() || "Phase complete", exitCode === 0 ? "success" : "warning");
    },
  });
}
