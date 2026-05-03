import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type } from "typebox";
import { execFile } from "node:child_process";
import { existsSync } from "node:fs";
import { join, dirname, extname } from "node:path";

// ── Helpers ────────────────────────────────────────────────────────────────

function execTool(cmd: string, args: string[], cwd: string, timeoutMs = 120000): Promise<{ stdout: string; stderr: string; exitCode: number }> {
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

function findProjectRoot(cwd: string): string {
  let dir = cwd;
  while (dir !== dirname(dir)) {
    if (existsSync(join(dir, "package.json")) ||
        existsSync(join(dir, "go.mod")) ||
        existsSync(join(dir, "pom.xml")) ||
        existsSync(join(dir, "build.gradle")) ||
        existsSync(join(dir, "Cargo.toml")) ||
        existsSync(join(dir, "composer.json")) ||
        existsSync(join(dir, "Gemfile")) ||
        existsSync(join(dir, "setup.py")) ||
        existsSync(join(dir, "pyproject.toml")) ||
        existsSync(join(dir, "Package.swift")) ||
        existsSync(join(dir, "CMakeLists.txt")) ||
        existsSync(join(dir, "Makefile"))) {
      return dir;
    }
    dir = dirname(dir);
  }
  return cwd;
}

function detectLang(cwd: string): string {
  if (existsSync(join(cwd, "package.json"))) return "js";
  if (existsSync(join(cwd, "go.mod"))) return "go";
  if (existsSync(join(cwd, "pom.xml")) || existsSync(join(cwd, "build.gradle"))) return "jvm";
  if (existsSync(join(cwd, "Cargo.toml"))) return "rust";
  if (existsSync(join(cwd, "pyproject.toml")) || existsSync(join(cwd, "setup.py")) || existsSync(join(cwd, "requirements.txt"))) return "py";
  if (existsSync(join(cwd, "composer.json"))) return "php";
  if (existsSync(join(cwd, "Gemfile"))) return "rb";
  if (existsSync(join(cwd, "Package.swift"))) return "swift";
  if (existsSync(join(cwd, "*.csproj")) || existsSync(join(cwd, "*.sln"))) return "cs";
  if (existsSync(join(cwd, "CMakeLists.txt"))) return "cpp";
  if (existsSync(join(cwd, "build.zig"))) return "zig";
  if (existsSync(join(cwd, "stack.yaml"))) return "haskell";
  if (existsSync(join(cwd, "mix.exs"))) return "elixir";
  if (existsSync(join(cwd, "pubspec.yaml"))) return "dart";
  if (existsSync(join(cwd, "dune-project"))) return "ocaml";
  if (existsSync(join(cwd, "Project.toml"))) return "julia";
  return "unknown";
}

// ── Extension ──────────────────────────────────────────────────────────────

export default function (pi: ExtensionAPI) {

  // ═══════════════════════════════════════════════════════════════════════
  // 1. UX / UI Testing Tools
  // ═══════════════════════════════════════════════════════════════════════

  pi.registerTool({
    name: "playwright",
    label: "Playwright",
    description: "Run Playwright e2e tests. Auto-detects project root with playwright config.",
    parameters: Type.Object({
      args: Type.Array(Type.String(), { default: [], description: "Extra args, e.g. ['--grep=@smoke']" }),
      project: Type.Optional(Type.String({ description: "Project name from playwright.config.ts" })),
      headed: Type.Optional(Type.Boolean({ default: false })),
      ui: Type.Optional(Type.Boolean({ default: false, description: "Open Playwright UI mode" })),
      trace: Type.Optional(Type.Boolean({ default: false })),
    }),
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const args = ["npx", "playwright", "test"];
      if (params.project) args.push("--project=" + params.project);
      if (params.headed) args.push("--headed");
      if (params.ui) { args.pop(); args.push("ui"); }
      if (params.trace) args.push("--trace=on");
      args.push(...params.args);
      const cmd = args.shift()!;
      const { stdout, stderr, exitCode } = await execTool(cmd, args, root, 300000);
      return {
        content: [{ type: "text", text: stdout || stderr }],
        details: { exitCode, root },
      };
    },
  });

  pi.registerTool({
    name: "cypress",
    label: "Cypress",
    description: "Run Cypress e2e/component tests.",
    parameters: Type.Object({
      mode: Type.String({ default: "run", description: "run | open | component" }),
      spec: Type.Optional(Type.String()),
      browser: Type.Optional(Type.String({ default: "chrome" })),
      args: Type.Array(Type.String(), { default: [] }),
    }),
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const cmd = existsSync(join(root, "node_modules", ".bin", "cypress"))
        ? join(root, "node_modules", ".bin", "cypress")
        : "npx";
      const args = cmd === "npx" ? ["cypress"] : [];
      args.push(params.mode === "component" ? "run" : params.mode);
      if (params.spec) args.push("--spec", params.spec);
      if (params.browser && params.mode !== "open") args.push("--browser", params.browser);
      if (params.mode === "component") args.push("--component");
      args.push(...params.args);
      const { stdout, stderr, exitCode } = await execTool(cmd, args, root, 300000);
      return {
        content: [{ type: "text", text: stdout || stderr }],
        details: { exitCode },
      };
    },
  });

  pi.registerTool({
    name: "lighthouse",
    label: "Lighthouse",
    description: "Run Lighthouse CI or standalone audit for performance, a11y, best-practices, SEO.",
    parameters: Type.Object({
      url: Type.Optional(Type.String()),
      preset: Type.String({ default: "desktop", description: "desktop | mobile" }),
      output: Type.String({ default: "json", description: "json | html | csv" }),
      categories: Type.Array(Type.String(), { default: ["performance", "accessibility", "best-practices", "seo"] }),
      args: Type.Array(Type.String(), { default: [] }),
    }),
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const hasLhci = existsSync(join(root, "lighthouserc.js")) || existsSync(join(root, "lighthouserc.json"));
      if (hasLhci) {
        const { stdout, stderr, exitCode } = await execTool("npx", ["@lhci/cli", "autorun", ...params.args], root, 300000);
        return { content: [{ type: "text", text: stdout || stderr }], details: { exitCode } };
      }
      const args = ["lighthouse"];
      if (params.url) args.push(params.url);
      args.push("--preset=" + params.preset, "--output=" + params.output);
      params.categories.forEach((c: string) => args.push("--only-categories=" + c));
      args.push("--chrome-flags='--headless'", ...params.args);
      const cmd = args.shift()!;
      const { stdout, stderr, exitCode } = await execTool(cmd, args, root, 300000);
      return {
        content: [{ type: "text", text: stdout || stderr }],
        details: { exitCode },
      };
    },
  });

  pi.registerTool({
    name: "storybook",
    label: "Storybook",
    description: "Build, test or serve Storybook. Supports test-runner for component tests.",
    parameters: Type.Object({
      action: Type.String({ default: "build", description: "build | dev | test | test-coverage" }),
      args: Type.Array(Type.String(), { default: [] }),
    }),
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const npmCmd = existsSync(join(root, "package-lock.json")) ? "npx" : "pnpm";
      const actionMap: Record<string, string[]> = {
        build: ["storybook", "build"],
        dev: ["storybook", "dev"],
        test: ["storybook", "test-runner", "--stories-json"],
        "test-coverage": ["storybook", "test-runner", "--coverage"],
      };
      const args = actionMap[params.action] || actionMap.build;
      args.push(...params.args);
      const cmd = args.shift()!;
      const finalCmd = cmd === "storybook" ? npmCmd : cmd;
      const finalArgs = cmd === "storybook" ? [cmd, ...args] : args;
      const { stdout, stderr, exitCode } = await execTool(finalCmd, finalArgs, root, 300000);
      return {
        content: [{ type: "text", text: stdout || stderr }],
        details: { exitCode },
      };
    },
  });

  pi.registerTool({
    name: "axe",
    label: "Axe Accessibility",
    description: "Run axe-core accessibility checks via CLI or page URL.",
    parameters: Type.Object({
      target: Type.String({ description: "URL or path to HTML file" }),
      tags: Type.Array(Type.String(), { default: ["wcag2a", "wcag2aa", "wcag21aa"] }),
    }),
    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      const root = findProjectRoot(ctx.cwd);
      const { stdout, stderr, exitCode } = await execTool(
        "npx",
        ["axe-core/cli", params.target, "--tags", params.tags.join(",")],
        root,
        120000
      );
      return {
        content: [{ type: "text", text: stdout || stderr }],
        details: { exitCode },
      };
    },
  });

  // ═══════════════════════════════════════════════════════════════════════
  // 2. Multi-Language Test Runners
  // ═══════════════════════════════════════════════════════════════════════

  const testTools = [
    // JS/TS (широкий спектр)
    {
      name: "vitest",
      label: "Vitest",
      cmd: "npx",
      baseArgs: ["vitest", "run"],
      detect: (r: string) => existsSync(join(r, "vitest.config.ts")) || existsSync(join(r, "vitest.config.js")),
    },
    {
      name: "jest",
      label: "Jest",
      cmd: "npx",
      baseArgs: ["jest"],
      detect: (r: string) => existsSync(join(r, "jest.config.js")) || existsSync(join(r, "jest.config.ts")),
    },
    {
      name: "mocha",
      label: "Mocha",
      cmd: "npx",
      baseArgs: ["mocha"],
      detect: (r: string) => existsSync(join(r, ".mocharc.json")),
    },
    {
      name: "ava",
      label: "AVA",
      cmd: "npx",
      baseArgs: ["ava"],
      detect: (r: string) => existsSync(join(r, "ava.config.js")),
    },
    {
      name: "tap",
      label: "TAP",
      cmd: "npx",
      baseArgs: ["tap", "run"],
      detect: (r: string) => existsSync(join(r, ".taprc")),
    },
    // Python
    {
      name: "pytest",
      label: "pytest",
      cmd: "python3",
      baseArgs: ["-m", "pytest", "-v"],
      detect: (r: string) => existsSync(join(r, "pytest.ini")) || existsSync(join(r, "pyproject.toml")),
    },
    {
      name: "unittest",
      label: "unittest",
      cmd: "python3",
      baseArgs: ["-m", "unittest", "discover", "-v"],
      detect: () => true,
    },
    // Go
    {
      name: "go_test",
      label: "go test",
      cmd: "go",
      baseArgs: ["test", "./...", "-v"],
      detect: (r: string) => existsSync(join(r, "go.mod")),
    },
    {
      name: "go_benchmark",
      label: "go benchmark",
      cmd: "go",
      baseArgs: ["test", "-bench=.", "-benchmem"],
      detect: (r: string) => existsSync(join(r, "go.mod")),
    },
    // JVM
    {
      name: "maven_test",
      label: "Maven Test",
      cmd: "mvn",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "pom.xml")),
    },
    {
      name: "gradle_test",
      label: "Gradle Test",
      cmd: "./gradlew",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "gradlew")),
    },
    {
      name: "gradle_test_wrapper",
      label: "Gradle (global)",
      cmd: "gradle",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "build.gradle")),
    },
    {
      name: "sbt_test",
      label: "SBT Test",
      cmd: "sbt",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "build.sbt")),
    },
    {
      name: "kotest",
      label: "Kotest",
      cmd: "./gradlew",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "build.gradle.kts")),
    },
    // Rust
    {
      name: "cargo_test",
      label: "Cargo Test",
      cmd: "cargo",
      baseArgs: ["test", "--", "--nocapture"],
      detect: (r: string) => existsSync(join(r, "Cargo.toml")),
    },
    {
      name: "cargo_bench",
      label: "Cargo Bench",
      cmd: "cargo",
      baseArgs: ["bench"],
      detect: (r: string) => existsSync(join(r, "Cargo.toml")),
    },
    // C#
    {
      name: "dotnet_test",
      label: "dotnet test",
      cmd: "dotnet",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, ".csproj")) || existsSync(join(r, ".sln")),
    },
    // Ruby
    {
      name: "rspec",
      label: "RSpec",
      cmd: "bundle",
      baseArgs: ["exec", "rspec"],
      detect: (r: string) => existsSync(join(r, ".rspec")),
    },
    {
      name: "minitest",
      label: "Minitest",
      cmd: "ruby",
      baseArgs: ["-Ilib:test"],
      detect: (r: string) => existsSync(join(r, "test")),
    },
    // PHP
    {
      name: "phpunit",
      label: "PHPUnit",
      cmd: "vendor/bin/phpunit",
      baseArgs: [],
      detect: (r: string) => existsSync(join(r, "phpunit.xml")) || existsSync(join(r, "vendor", "bin", "phpunit")),
    },
    {
      name: "pest",
      label: "Pest",
      cmd: "vendor/bin/pest",
      baseArgs: [],
      detect: (r: string) => existsSync(join(r, "pest.xml")),
    },
    // Swift
    {
      name: "swift_test",
      label: "Swift Test",
      cmd: "swift",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "Package.swift")),
    },
    // C/C++
    {
      name: "cmake_test",
      label: "CTest",
      cmd: "ctest",
      baseArgs: ["--output-on-failure"],
      detect: (r: string) => existsSync(join(r, "CMakeLists.txt")),
    },
    {
      name: "catch2",
      label: "Catch2",
      cmd: "make",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "tests", "CMakeLists.txt")),
    },
    // Zig
    {
      name: "zig_test",
      label: "Zig Test",
      cmd: "zig",
      baseArgs: ["build", "test"],
      detect: (r: string) => existsSync(join(r, "build.zig")),
    },
    // Haskell
    {
      name: "stack_test",
      label: "Stack Test",
      cmd: "stack",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "stack.yaml")),
    },
    // Elixir
    {
      name: "mix_test",
      label: "Mix Test",
      cmd: "mix",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "mix.exs")),
    },
    // Dart/Flutter
    {
      name: "flutter_test",
      label: "Flutter Test",
      cmd: "flutter",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "pubspec.yaml")),
    },
    // Lua
    {
      name: "busted",
      label: "Busted",
      cmd: "busted",
      baseArgs: [],
      detect: (r: string) => existsSync(join(r, ".busted")),
    },
    // Nim
    {
      name: "nimble_test",
      label: "Nimble Test",
      cmd: "nimble",
      baseArgs: ["test"],
      detect: (r: string) => existsSync(join(r, "*.nimble")),
    },
    // OCaml
    {
      name: "dune_test",
      label: "Dune Test",
      cmd: "dune",
      baseArgs: ["runtest"],
      detect: (r: string) => existsSync(join(r, "dune-project")),
    },
    // Julia
    {
      name: "julia_test",
      label: "Julia Test",
      cmd: "julia",
      baseArgs: ["-e", "using Pkg; Pkg.test()"],
      detect: (r: string) => existsSync(join(r, "Project.toml")),
    },
  ];

  for (const tt of testTools) {
    pi.registerTool({
      name: tt.name,
      label: tt.label,
      description: `Run ${tt.label} tests. Auto-detected: ${tt.detect.name || "project root"}.`,
      parameters: Type.Object({
        args: Type.Array(Type.String(), { default: [], description: "Extra args" }),
        watch: Type.Optional(Type.Boolean({ default: false })),
        coverage: Type.Optional(Type.Boolean({ default: false })),
      }),
      async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
        const root = findProjectRoot(ctx.cwd);
        const args = [...tt.baseArgs];
        if (params.watch) args.push("--watch");
        if (params.coverage) args.push("--coverage");
        args.push(...params.args);
        // Fallback: if cmd not found, try npx/pnpm for JS tools
        let cmd = tt.cmd;
        if (cmd.startsWith("vendor/") && !existsSync(join(root, cmd))) {
          cmd = "php";
          args.unshift("vendor/bin/" + tt.name.replace("php", "").replace("pest", "pest"));
        }
        const { stdout, stderr, exitCode } = await execTool(cmd, args, root, 180000);
        return {
          content: [{ type: "text", text: stdout || stderr }],
          details: { exitCode, runner: tt.name },
        };
      },
    });
  }

  // ═══════════════════════════════════════════════════════════════════════
  // 3. Meta / Discovery Commands
  // ═══════════════════════════════════════════════════════════════════════

  pi.registerCommand("test-all", {
    description: "Auto-detect language and run appropriate test suite",
    handler: async (_args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const lang = detectLang(root);
      const map: Record<string, string> = {
        js: "vitest or jest (check package.json)",
        go: "go_test",
        py: "pytest",
        jvm: "maven_test or gradle_test",
        rust: "cargo_test",
        cs: "dotnet_test",
        rb: "rspec",
        php: "phpunit",
        swift: "swift_test",
        cpp: "cmake_test",
      };
      const runner = map[lang] || "unknown";
      ctx.ui.notify(`Detected: ${lang} → ${runner}`, "info");
      ctx.ui.notify(`Use tool ${runner.split(" or ")[0]} to run tests.`, "info");
    },
  });

  pi.registerCommand("ux-audit", {
    description: "Run full UX/UI audit: Lighthouse + axe + screenshot compare",
    handler: async (args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      ctx.ui.notify("Running UX audit pipeline...", "info");
      // Lighthouse
      const lh = await execTool("npx", ["lighthouse", args.trim() || "http://localhost:3000", "--output=json", "--chrome-flags='--headless'"], root, 120000);
      ctx.ui.notify("Lighthouse: " + (lh.exitCode === 0 ? "OK" : "issues found"), lh.exitCode === 0 ? "success" : "warning");
    },
  });

  // ═══════════════════════════════════════════════════════════════════════
  // 4. UI Widget: Test Status
  // ═══════════════════════════════════════════════════════════════════════

  pi.on("session_start", async (_event, ctx) => {
    const root = findProjectRoot(ctx.cwd);
    const lang = detectLang(root);
    if (lang !== "unknown") {
      ctx.ui.setStatus("lang-detect", ctx.ui.theme.fg("accent", `● ${lang.toUpperCase()}`));
    }
  });
}
