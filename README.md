# LazyTest

**The terminal test runner that actually understands your monorepo.**

PHPUnit and Vitest in the same project? Backend tests behind Docker, frontend tests on bare metal? Different working directories, different file patterns, different everything? LazyTest handles all of it from a single TUI — no tab-switching, no re-running scripts, no context-switching between frameworks.

![LazyTest Demo](demo.gif?v=2)

---

**fzf-style fuzzy search** across 1,000+ test files. **Tab multi-select** to pick exactly what you want to run. **Real-time streaming** so you see results the instant they happen, not after the whole suite finishes. **Parallel execution** of multiple frameworks simultaneously. **Zero-config auto-detection** that just works when you drop into a directory.

Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). Single binary, no runtime dependencies.

---

## Why LazyTest?

Most test runners are built for one framework. In a monorepo you end up with a patchwork of shell scripts, Makefile targets, and muscle memory. "Was it `make test-php` or `npm run test`? Which directory am I in?"

LazyTest replaces that with one command:

```bash
lazytest
```

It scans your project, detects frameworks, and presents every test file in a searchable, selectable list — tagged with `[PHP]`, `[VT]`, or whatever your targets are. Pick files across different frameworks, hit Enter, and watch them run in parallel with live-updating results.

### What Sets It Apart

| | LazyTest | Typical CLI runner |
|-|----------|--------------------|
| **Monorepo** | First-class. Multiple targets with separate commands, dirs, and patterns. | One framework at a time. |
| **File selection** | fzf-like fuzzy search + Tab multi-select across all targets. | Glob patterns or "run everything". |
| **Feedback loop** | Streaming — see each test result as it finishes. | Wait for the entire run, then scroll. |
| **Setup** | Auto-detects `phpunit.xml`, `vitest.config.*`, and `jest.config.*`. Zero config to start. | Config per framework. |
| **Cross-framework** | PHPUnit + Vitest + Jest + pytest + anything with TeamCity/TAP output. | Framework-specific. |

## Features

### Fuzzy File Search

fzf-inspired two-phase fuzzy matching. Type a few characters and LazyTest finds your test files instantly — contiguous substring matches are ranked first, followed by subsequence matches with smart scoring. Matched characters are highlighted so you can see exactly what's matching.

The algorithm uses a backward pass to find the tightest match window, bonuses for word-boundary hits (`/`, `_`, `-`, `.`), and a span cutoff to eliminate scattered noise. Queries of 3+ characters require at least one consecutive character pair, so `tutor` matches `TutorAgent` but not `tests/Unit/Bookmark`.

### Multi-Select

Select individual files with `Tab` (cursor advances automatically, just like fzf), or batch-select with `Ctrl+A`. Selected files are marked with `◆` and the header shows the count. Press `Enter` to run only the selected files — or just hit `Enter` with no selection to run the file under your cursor.

### Multi-Framework Parallel Execution

Each target runs in its own goroutine. LazyTest groups selected files by target, spawns parallel processes, and merges their event streams into a single UI via fan-in. A PHPUnit suite running in Docker and a Vitest suite running locally execute simultaneously — and you see both updating in real time.

### Zero-Config Auto-Detection

Run `lazytest` with no config file. It walks up to 3 directory levels looking for `phpunit.xml`, `phpunit.xml.dist`, `vitest.config.{ts,mts,js}`, and `jest.config.{ts,js,mjs,cjs}`, then builds targets with sensible defaults. Nested monorepo structures are handled — each detected project becomes its own target with the correct working directory.

### Real-Time Streaming Results

No staring at a frozen terminal. Test results appear as they execute — each target shows a live tree of suites and test cases with status icons (`◉` running, `✓` passed, `✗` failed, `⊘` skipped) and durations. The moment one test finishes, you see it.

### Split-Pane Results View

After execution, results mode shows a two-pane layout: suite/test tree on the left, detailed information on the right. Navigate with vim keys (`h`/`j`/`k`/`l`), drill into failures to see messages and full stack traces, or press `f` to filter to failures only.

### Target Badges

`[PHP]` and `[VT]` badges (and custom badges for any target name) appear next to every file and result entry, so you always know which framework you're looking at — even when files from different targets are interleaved.

### Built-in Vitest & Jest Reporters

TeamCity-compatible reporters for Vitest and Jest are embedded in the binary via `go:embed`. No need to `npm install` separate reporter packages — LazyTest extracts them to temp files and injects the path through the `{reporter}` template variable.

### Editor Integration

Press `o` in results mode to open the relevant test file in your OS default application (`open` on macOS, `xdg-open` on Linux, `start` on Windows).

### TeamCity + TAP Parsing

The streaming parser auto-detects the output format on the first meaningful line. TeamCity service messages are the primary format, with TAP v13 as a fallback. Any test runner that speaks either protocol works with LazyTest.

## Install

### Pre-built binary (recommended)

Download the latest binary from the [Releases page](https://github.com/meijin/lazytest/releases/latest).

| OS                    | File                             |
|-----------------------|----------------------------------|
| macOS (Apple Silicon) | `lazytest_darwin_arm64.tar.gz`   |
| macOS (Intel)         | `lazytest_darwin_amd64.tar.gz`   |
| Linux (x86_64)        | `lazytest_linux_amd64.tar.gz`    |
| Linux (ARM64)         | `lazytest_linux_arm64.tar.gz`    |
| Windows (x86_64)      | `lazytest_windows_amd64.zip`     |
| Windows (ARM64)       | `lazytest_windows_arm64.zip`     |

```bash
tar xzf lazytest_darwin_arm64.tar.gz
mv lazytest /usr/local/bin/
```

### go install

```bash
go install github.com/meijin/lazytest/cmd/lazytest@latest
```

### Build from source

```bash
git clone https://github.com/meijin/lazytest.git
cd lazytest
go build -o lazytest ./cmd/lazytest
```

## Quick Start

Run `lazytest` in any directory containing `phpunit.xml` or `vitest.config.ts`:

```bash
lazytest
```

LazyTest auto-detects test frameworks and scans for test files. To use a config file in a custom location:

```bash
lazytest -config path/to/.lazytest.yml
```

## Configuration

Create a `.lazytest.yml` in your project root:

```yaml
targets:
  - name: phpunit
    command: "docker compose exec php-fpm ./vendor/bin/phpunit --teamcity {files}"
    test_dirs:
      - server/src/tests/
    file_pattern: "*Test.php"
    path_strip_prefix: "server/src/"
    working_dir: "server/"

  - name: vitest
    command: "npx vitest run --reporter={reporter} {files}"
    test_dirs:
      - client/next/src/
      - client/next/app/
    file_pattern: "*.test.ts,*.test.tsx"
    working_dir: "client/next/"
```

### Target Options

| Key                | Description |
|--------------------|-------------|
| `name`             | Target identifier. `"phpunit"` and `"vitest"` get smart defaults for all other fields. |
| `command`          | Command template. `{files}` is replaced with space-separated test file paths. `{file}` is replaced with the first file only. `{reporter}` is replaced with the path to the built-in Vitest reporter. |
| `test_dirs`        | Directories to scan for test files. |
| `file_pattern`     | Glob pattern(s) to match test files. Comma-separated for OR matching (e.g. `"*.test.ts,*.test.tsx"`). |
| `path_strip_prefix`| Prefix to strip from file paths before passing to the command. |
| `working_dir`      | Working directory for the command (relative to project root). File paths are auto-adjusted to be relative to this directory. |

### Defaults by Target Name

When a field is omitted, defaults are applied based on the target name:

| Target    | `file_pattern`                          | `command`                                              | `test_dirs` |
|-----------|------------------------------------------|--------------------------------------------------------|-------------|
| `phpunit` | `*Test.php`                              | `./vendor/bin/phpunit --teamcity {files}`              | `tests/`    |
| `vitest`  | `*.test.ts,*.test.tsx`                   | `npx vitest run --reporter={reporter} {files}`         | `src/`      |
| `jest`    | `*.test.ts,*.test.tsx,*.test.js,*.test.jsx` | `npx jest --reporters={reporter} -- {files}`           | `src/`      |

If no `.lazytest.yml` is found, LazyTest walks up to 3 directory levels to auto-detect `phpunit.xml`, `vitest.config.{ts,mts,js}`, and `jest.config.{ts,js,mjs,cjs}`.

## Key Bindings

### Search Mode

| Key                          | Action |
|------------------------------|--------|
| Type any text                | Fuzzy filter test files |
| `Tab`                        | Toggle selection on cursor file (moves cursor down) |
| `Ctrl+A`                     | Select all / deselect all filtered files |
| `Enter`                      | Run selected files (or cursor file if none selected) |
| `↑` / `Ctrl+P` / `Ctrl+K`   | Move cursor up |
| `↓` / `Ctrl+N` / `Ctrl+J`   | Move cursor down |
| `Ctrl+C`                     | Quit |

### Running Mode

| Key       | Action |
|-----------|--------|
| `Esc`     | Cancel run, return to search |
| `Ctrl+C`  | Quit |

### Results Mode

| Key              | Action |
|------------------|--------|
| `j` / `↓`       | Move down |
| `k` / `↑`       | Move up |
| `l`              | Focus detail pane |
| `h`              | Focus list pane |
| `f`              | Toggle failures only filter |
| `o`              | Open test file in OS default application |
| `r`              | Re-run same files |
| `R`              | Re-run all files |
| `Enter` / `Esc`  | Return to search |
| `q` / `Ctrl+C`   | Quit |

## Framework Setup

LazyTest works with any test runner that outputs [TeamCity service messages](https://www.jetbrains.com/help/teamcity/service-messages.html#Reporting+Tests) or [TAP v13](https://testanything.org/tap-version-13-specification.html).

**PHPUnit** — built-in `--teamcity` flag:
```yaml
targets:
  - name: phpunit
    command: "./vendor/bin/phpunit --teamcity {files}"
```

**Vitest** — uses the built-in reporter (no extra install needed):
```yaml
targets:
  - name: vitest
    command: "npx vitest run --reporter={reporter} {files}"
```

Or with an external reporter like [vitest-teamcity-reporter](https://www.npmjs.com/package/vitest-teamcity-reporter):
```yaml
targets:
  - name: vitest
    command: "npx vitest run --reporter=vitest-teamcity-reporter {files}"
```

**Pest (PHP)**:
```yaml
targets:
  - name: pest
    command: "./vendor/bin/pest --teamcity {files}"
    file_pattern: "*Test.php"
```

**Jest** — uses the built-in reporter (no extra install needed):
```yaml
targets:
  - name: jest
    command: "npx jest --reporters={reporter} -- {files}"
```

**pytest** with [teamcity-messages](https://pypi.org/project/teamcity-messages/):
```yaml
targets:
  - name: pytest
    command: "python -m pytest --teamcity {files}"
    file_pattern: "test_*.py"
```

## Architecture

```
cmd/lazytest/main.go    Entry point
internal/
  config/     Configuration loading (.lazytest.yml / framework auto-detection)
  discovery/  Test file scanning (glob pattern matching, multi-target)
  domain/     Domain types (TestFile, TestCase, TestSuite, TestRun, AggregatedRun)
  parser/     Streaming parser (auto-detects TeamCity / TAP format)
  reporter/   Built-in Vitest reporter (embedded via go:embed)
  runner/     Multi-target parallel execution (goroutine per target, fan-in)
  ui/         Bubble Tea UI (Search → Running → Results)
```

### Processing Flow

1. `config.Load()` reads `.lazytest.yml` or falls back to `DetectFrameworks()` auto-detection
2. `discovery.ScanAllTargets()` collects test files across all targets
3. **Search mode**: fuzzy filter and select files (with target badges)
4. **Enter** starts execution — `executor.Run()` groups files by target and spawns a goroutine per target
5. Each goroutine pipes stdout through `parser.ParseStream()`, wraps events in `TargetEvent`, and sends them to a shared channel
6. **Running mode**: UI consumes events in real time, updating per-target progress
7. On completion, `AggregatedRun` is built and **Results mode** displays the unified output

## License

MIT
