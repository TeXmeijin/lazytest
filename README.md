# lazytest

A TUI test runner for PHPUnit, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

lazytest parses [TeamCity format](https://www.jetbrains.com/help/teamcity/service-messages.html#Reporting+Tests) output in real time and displays results interactively in your terminal.

## Features

- **Fuzzy file search** - Quickly filter and select test files to run
- **Real-time streaming** - Watch test results appear as they execute
- **Split-pane results** - Browse suites/tests on the left, details on the right
- **Failure details** - View failure messages and stack traces inline
- **Re-run from results** - Press `r` to re-run without leaving the TUI
- **Editor integration** - Press `o` to open the failing test in your editor
- **Auto-detection** - Picks up `phpunit.xml` / `phpunit.xml.dist` automatically

## Install

```bash
go install github.com/meijin/lazytest/cmd/lazytest@latest
```

Or build from source:

```bash
git clone https://github.com/meijin/lazytest.git
cd lazytest
go build -o lazytest ./cmd/lazytest
```

## Quick Start

Run `lazytest` in any directory containing `phpunit.xml`:

```bash
lazytest
```

That's it. lazytest auto-detects your PHPUnit configuration and scans for test files.

## Configuration

Create a `.lazytest.yml` in your project root for custom settings:

```yaml
command: "./vendor/bin/phpunit --teamcity {files}"
test_dirs:
  - tests/
file_pattern: "*Test.php"
path_strip_prefix: ""
editor: "code"
```

| Key                | Default                                  | Description                                        |
|--------------------|------------------------------------------|----------------------------------------------------|
| `command`          | `./vendor/bin/phpunit --teamcity {files}` | Command template. `{files}` and `{file}` are replaced with test file paths. |
| `test_dirs`        | `["tests/"]`                             | Directories to scan for test files                 |
| `file_pattern`     | `*Test.php`                              | Glob pattern to match test files                   |
| `path_strip_prefix`| `""`                                     | Prefix to strip from file paths before passing to the command |
| `editor`           | `$LAZYTEST_EDITOR` or `zed`              | Editor command for opening test files              |

If no `.lazytest.yml` is found, lazytest falls back to parsing `phpunit.xml` / `phpunit.xml.dist` for test directories.

## Key Bindings

### Search Mode

| Key       | Action                   |
|-----------|--------------------------|
| `↑` / `Ctrl+P` | Move cursor up     |
| `↓` / `Ctrl+N` | Move cursor down   |
| `Enter`   | Run filtered tests       |
| `Ctrl+A`  | Run all tests            |
| `Tab`     | Switch to results        |
| `Ctrl+C`  | Quit                     |

### Running Mode

| Key       | Action          |
|-----------|-----------------|
| `Esc`     | Cancel run      |
| `Ctrl+C`  | Quit            |

### Results Mode

| Key       | Action                |
|-----------|-----------------------|
| `j` / `↓` | Move down            |
| `k` / `↑` | Move up              |
| `l`       | Focus detail pane     |
| `h`       | Focus list pane       |
| `r`       | Re-run same tests     |
| `R`       | Re-run all tests      |
| `f`       | Toggle failures only  |
| `o`       | Open in editor        |
| `Enter` / `Esc` | Back to search |
| `q`       | Quit                  |

## Using with Other Frameworks

lazytest works with any test runner that outputs [TeamCity service messages](https://www.jetbrains.com/help/teamcity/service-messages.html#Reporting+Tests). Configure the `command`, `test_dirs`, and `file_pattern` in `.lazytest.yml`:

**Pest (PHP)**
```yaml
command: "./vendor/bin/pest --teamcity {files}"
test_dirs: ["tests/"]
file_pattern: "*Test.php"
```

**Jest (JavaScript/TypeScript)** with [jest-teamcity](https://www.npmjs.com/package/jest-teamcity):
```yaml
command: "npx jest --reporters=jest-teamcity {files}"
test_dirs: ["__tests__/", "src/"]
file_pattern: "*.test.ts"
```

**pytest (Python)** with [teamcity-messages](https://pypi.org/project/teamcity-messages/):
```yaml
command: "python -m pytest --teamcity {files}"
test_dirs: ["tests/"]
file_pattern: "test_*.py"
```

## Architecture

```
cmd/lazytest/main.go    Entry point
internal/
  config/     Configuration loading (.lazytest.yml / phpunit.xml auto-detection)
  discovery/  Test file scanning (glob pattern matching)
  domain/     Domain types (TestStatus, TestCase, TestSuite, TestRun, TestFile)
  parser/     TeamCity format streaming parser
  runner/     Test execution (command template expansion + goroutines)
  ui/         Bubble Tea UI (SearchMode -> RunningMode -> ResultsMode)
```

## License

MIT
