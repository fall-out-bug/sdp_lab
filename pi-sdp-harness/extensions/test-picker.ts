import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { DynamicBorder } from "@mariozechner/pi-coding-agent";
import {
  Container,
  type SelectItem,
  SelectList,
  Text,
  matchesKey,
  Key,
} from "@mariozechner/pi-tui";
import { execFile } from "node:child_process";
import { existsSync } from "node:fs";
import { join, dirname } from "node:path";

// ── Helpers ────────────────────────────────────────────────────────────────

function findProjectRoot(cwd: string): string {
  let dir = cwd;
  while (dir !== dirname(dir)) {
    if (
      existsSync(join(dir, "package.json")) ||
      existsSync(join(dir, "go.mod")) ||
      existsSync(join(dir, "pom.xml")) ||
      existsSync(join(dir, "build.gradle")) ||
      existsSync(join(dir, "Cargo.toml")) ||
      existsSync(join(dir, "pyproject.toml")) ||
      existsSync(join(dir, "composer.json")) ||
      existsSync(join(dir, "Gemfile")) ||
      existsSync(join(dir, "Package.swift"))
    ) {
      return dir;
    }
    dir = dirname(dir);
  }
  return cwd;
}

function detectLang(root: string): string {
  if (existsSync(join(root, "package.json"))) return "js";
  if (existsSync(join(root, "go.mod"))) return "go";
  if (existsSync(join(root, "pom.xml")) || existsSync(join(root, "build.gradle"))) return "jvm";
  if (existsSync(join(root, "Cargo.toml"))) return "rust";
  if (existsSync(join(root, "pyproject.toml")) || existsSync(join(root, "setup.py"))) return "py";
  if (existsSync(join(root, "composer.json"))) return "php";
  if (existsSync(join(root, "Gemfile"))) return "ruby";
  if (existsSync(join(root, "Package.swift"))) return "swift";
  if (existsSync(join(root, ".csproj")) || existsSync(join(root, ".sln"))) return "cs";
  if (existsSync(join(root, "CMakeLists.txt"))) return "cpp";
  return "unknown";
}

interface RunnerDef {
  name: string;
  label: string;
  cmd: string;
  args: string[];
}

const RUNNERS: Record<string, RunnerDef[]> = {
  js: [
    { name: "vitest", label: "Vitest", cmd: "npx", args: ["vitest", "run"] },
    { name: "jest", label: "Jest", cmd: "npx", args: ["jest"] },
    { name: "mocha", label: "Mocha", cmd: "npx", args: ["mocha"] },
    { name: "playwright", label: "Playwright E2E", cmd: "npx", args: ["playwright", "test"] },
    { name: "cypress", label: "Cypress", cmd: "npx", args: ["cypress", "run"] },
  ],
  go: [
    { name: "go_test", label: "go test", cmd: "go", args: ["test", "./...", "-v"] },
    { name: "go_benchmark", label: "go benchmark", cmd: "go", args: ["test", "-bench=.", "-benchmem"] },
  ],
  py: [
    { name: "pytest", label: "pytest", cmd: "python3", args: ["-m", "pytest", "-v"] },
    { name: "unittest", label: "unittest", cmd: "python3", args: ["-m", "unittest", "discover", "-v"] },
  ],
  jvm: [
    { name: "maven_test", label: "Maven Test", cmd: "mvn", args: ["test"] },
    { name: "gradle_test", label: "Gradle Test", cmd: "./gradlew", args: ["test"] },
    { name: "sbt_test", label: "SBT Test", cmd: "sbt", args: ["test"] },
  ],
  rust: [
    { name: "cargo_test", label: "Cargo Test", cmd: "cargo", args: ["test"] },
    { name: "cargo_bench", label: "Cargo Bench", cmd: "cargo", args: ["bench"] },
  ],
  cs: [
    { name: "dotnet_test", label: "dotnet test", cmd: "dotnet", args: ["test"] },
  ],
  ruby: [
    { name: "rspec", label: "RSpec", cmd: "bundle", args: ["exec", "rspec"] },
    { name: "minitest", label: "Minitest", cmd: "ruby", args: ["-Ilib:test"] },
  ],
  php: [
    { name: "phpunit", label: "PHPUnit", cmd: "vendor/bin/phpunit", args: [] },
    { name: "pest", label: "Pest", cmd: "vendor/bin/pest", args: [] },
  ],
  swift: [
    { name: "swift_test", label: "Swift Test", cmd: "swift", args: ["test"] },
  ],
  cpp: [
    { name: "cmake_test", label: "CTest", cmd: "ctest", args: ["--output-on-failure"] },
  ],
  zig: [
    { name: "zig_test", label: "Zig Test", cmd: "zig", args: ["build", "test"] },
  ],
  haskell: [
    { name: "stack_test", label: "Stack Test", cmd: "stack", args: ["test"] },
  ],
  elixir: [
    { name: "mix_test", label: "Mix Test", cmd: "mix", args: ["test"] },
  ],
  kotlin: [
    { name: "gradle_test", label: "Gradle Test", cmd: "./gradlew", args: ["test"] },
  ],
  dart: [
    { name: "flutter_test", label: "Flutter Test", cmd: "flutter", args: ["test"] },
  ],
  lua: [
    { name: "busted", label: "Busted", cmd: "busted", args: [] },
  ],
  nim: [
    { name: "nimble_test", label: "Nimble Test", cmd: "nimble", args: ["test"] },
  ],
  ocaml: [
    { name: "dune_test", label: "Dune Test", cmd: "dune", args: ["runtest"] },
  ],
  julia: [
    { name: "julia_test", label: "Julia Test", cmd: "julia", args: ["-e", "using Pkg; Pkg.test()"] },
  ],
};

const LANG_NAMES: Record<string, string> = {
  js: "JavaScript / TypeScript",
  go: "Go",
  py: "Python",
  jvm: "JVM (Java / Kotlin / Scala)",
  rust: "Rust",
  cs: "C# / .NET",
  ruby: "Ruby",
  php: "PHP",
  swift: "Swift",
  cpp: "C / C++",
  zig: "Zig",
  haskell: "Haskell",
  elixir: "Elixir / Erlang",
  kotlin: "Kotlin Multiplatform",
  dart: "Dart / Flutter",
  lua: "Lua",
  nim: "Nim",
  ocaml: "OCaml",
  julia: "Julia",
};

// ── Extension ──────────────────────────────────────────────────────────────

export default function (pi: ExtensionAPI) {
  pi.registerCommand("test-picker", {
    description: "Interactive test runner picker (TUI overlay)",
    handler: async (_args, ctx) => {
      const root = findProjectRoot(ctx.cwd);
      const detected = detectLang(root);

      // Step 1: Language selection
      const langItems: SelectItem[] = Object.entries(LANG_NAMES).map(
        ([key, label]) => ({
          value: key,
          label,
          description: key === detected ? "detected" : undefined,
        })
      );

      const selectedLang = await ctx.ui.custom<string | null>(
        (tui, theme, _kb, done) => {
          const container = new Container();
          container.addChild(
            new DynamicBorder((s: string) => theme.fg("accent", s))
          );
          container.addChild(
            new Text(
              theme.fg("accent", theme.bold("Select Language")),
              1,
              0
            )
          );
          const list = new SelectList(langItems, Math.min(langItems.length, 10), {
            selectedPrefix: (t) => theme.fg("accent", t),
            selectedText: (t) => theme.fg("accent", t),
            description: (t) => theme.fg("muted", t),
            scrollInfo: (t) => theme.fg("dim", t),
          });
          list.onSelect = (item) => done(item.value);
          list.onCancel = () => done(null);
          container.addChild(list);
          container.addChild(
            new Text(
              theme.fg("dim", "↑↓ navigate • enter select • esc cancel"),
              1,
              0
            )
          );
          container.addChild(
            new DynamicBorder((s: string) => theme.fg("accent", s))
          );
          return {
            render: (w) => container.render(w),
            invalidate: () => container.invalidate(),
            handleInput: (data) => {
              list.handleInput(data);
              tui.requestRender();
            },
          };
        },
        { overlay: true }
      );

      if (!selectedLang) {
        ctx.ui.notify("Cancelled", "info");
        return;
      }

      // Step 2: Runner selection
      const runners = RUNNERS[selectedLang] || [];
      if (runners.length === 0) {
        ctx.ui.notify(`No runners for ${selectedLang}`, "warning");
        return;
      }

      const runnerItems: SelectItem[] = runners.map((r) => ({
        value: r.name,
        label: r.label,
        description: `${r.cmd} ${r.args.join(" ")}`,
      }));

      const selectedRunner = await ctx.ui.custom<string | null>(
        (tui, theme, _kb, done) => {
          const container = new Container();
          container.addChild(
            new DynamicBorder((s: string) => theme.fg("accent", s))
          );
          container.addChild(
            new Text(
              theme.fg("accent", theme.bold(`Select Runner (${LANG_NAMES[selectedLang]})`)),
              1,
              0
            )
          );
          const list = new SelectList(runnerItems, Math.min(runnerItems.length, 8), {
            selectedPrefix: (t) => theme.fg("accent", t),
            selectedText: (t) => theme.fg("accent", t),
            description: (t) => theme.fg("muted", t),
            scrollInfo: (t) => theme.fg("dim", t),
          });
          list.onSelect = (item) => done(item.value);
          list.onCancel = () => done(null);
          container.addChild(list);
          container.addChild(
            new Text(
              theme.fg("dim", "↑↓ navigate • enter select • esc cancel"),
              1,
              0
            )
          );
          container.addChild(
            new DynamicBorder((s: string) => theme.fg("accent", s))
          );
          return {
            render: (w) => container.render(w),
            invalidate: () => container.invalidate(),
            handleInput: (data) => {
              list.handleInput(data);
              tui.requestRender();
            },
          };
        },
        { overlay: true }
      );

      if (!selectedRunner) {
        ctx.ui.notify("Cancelled", "info");
        return;
      }

      const runner = runners.find((r) => r.name === selectedRunner)!;

      // Step 3: Options
      const optsItems: SelectItem[] = [
        { value: "run", label: "Run tests", description: "default" },
        { value: "coverage", label: "Run with coverage", description: "--coverage" },
        { value: "watch", label: "Watch mode", description: "--watch" },
      ];

      const selectedOpt = await ctx.ui.custom<string | null>(
        (tui, theme, _kb, done) => {
          const container = new Container();
          container.addChild(
            new DynamicBorder((s: string) => theme.fg("accent", s))
          );
          container.addChild(
            new Text(
              theme.fg("accent", theme.bold("Options")),
              1,
              0
            )
          );
          const list = new SelectList(optsItems, optsItems.length, {
            selectedPrefix: (t) => theme.fg("accent", t),
            selectedText: (t) => theme.fg("accent", t),
            description: (t) => theme.fg("muted", t),
          });
          list.onSelect = (item) => done(item.value);
          list.onCancel = () => done("run");
          container.addChild(list);
          container.addChild(
            new DynamicBorder((s: string) => theme.fg("accent", s))
          );
          return {
            render: (w) => container.render(w),
            invalidate: () => container.invalidate(),
            handleInput: (data) => {
              list.handleInput(data);
              tui.requestRender();
            },
          };
        },
        { overlay: true }
      );

      // Build final command
      const args = [...runner.args];
      if (selectedOpt === "coverage") {
        if (selectedLang === "js") args.push("--coverage");
        else if (selectedLang === "py") args.push("--cov");
        else if (selectedLang === "go") args.push("-coverprofile=coverage.out");
        else if (selectedLang === "jvm") args.push("-Djacoco.enabled=true");
      }
      if (selectedOpt === "watch") {
        if (selectedLang === "js") args.push("--watch");
        else if (selectedLang === "go") args.push("-watch"); // not native, just example
      }

      // Execute
      ctx.ui.notify(`Running: ${runner.cmd} ${args.join(" ")}`, "info");
      execFile(
        runner.cmd,
        args,
        { cwd: root, timeout: 300000 },
        (error, stdout, stderr) => {
          const exitCode = error ? (error as any).code || 1 : 0;
          const output = stdout || stderr;
          ctx.ui.notify(
            exitCode === 0 ? "Tests passed ✓" : "Tests failed ✗",
            exitCode === 0 ? "success" : "error"
          );
          // Print full output as a "system" message
          ctx.ui.notify(output.slice(0, 500), exitCode === 0 ? "info" : "error");
        }
      );
    },
  });
}
