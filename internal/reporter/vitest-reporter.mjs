/**
 * LazyTest custom Vitest reporter - TeamCity streaming output.
 * Emits TeamCity service messages on each test case result for real-time parsing.
 * Compatible with Vitest v2+ (Reported Tasks API).
 */

function escape(str) {
  if (!str) return "";
  return str
    .replace(/\|/g, "||")
    .replace(/'/g, "|'")
    .replace(/\n/g, "|n")
    .replace(/\r/g, "|r")
    .replace(/\[/g, "|[")
    .replace(/]/g, "|]");
}

function msg(type, attrs) {
  const parts = Object.entries(attrs)
    .map(([k, v]) => `${k}='${escape(String(v))}'`)
    .join(" ");
  process.stdout.write(`##teamcity[${type} ${parts}]\n`);
}

function getSuiteName(testCase) {
  const parts = [];
  let node = testCase.parent;
  while (node) {
    if (node.type === "suite") {
      parts.unshift(node.name);
    }
    // Stop at module level
    if (node.type === "module") {
      break;
    }
    node = node.parent;
  }
  if (parts.length > 0) {
    return parts.join(" > ");
  }
  // Fallback to module file name
  const mod = testCase.module;
  if (mod && mod.moduleId) {
    return mod.moduleId.split("/").pop();
  }
  return "unknown";
}

export default class TeamCityStreamingReporter {
  #currentSuite = null;

  onTestCaseResult(testCase) {
    const result = testCase.result();
    const name = testCase.name;
    const suiteName = getSuiteName(testCase);

    // Handle suite transitions
    if (this.#currentSuite !== suiteName) {
      if (this.#currentSuite !== null) {
        msg("testSuiteFinished", { name: this.#currentSuite });
      }
      msg("testSuiteStarted", { name: suiteName });
      this.#currentSuite = suiteName;
    }

    msg("testStarted", { name });

    if (result.state === "failed") {
      const error = result.errors?.[0];
      const message = error?.message || "Test failed";
      const details = error?.stack || "";
      msg("testFailed", { name, message, details });
    } else if (result.state === "skipped") {
      msg("testIgnored", { name, message: "skipped" });
    }

    const duration = result.duration || 0;
    msg("testFinished", { name, duration });
  }

  onTestRunEnd() {
    if (this.#currentSuite !== null) {
      msg("testSuiteFinished", { name: this.#currentSuite });
      this.#currentSuite = null;
    }
  }
}
