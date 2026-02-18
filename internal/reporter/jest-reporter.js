/**
 * LazyTest custom Jest reporter - TeamCity streaming output.
 * Emits TeamCity service messages per test result for real-time parsing.
 * Compatible with Jest 27+.
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

class LazyTestJestReporter {
  onTestResult(_test, testResult) {
    const filePath = testResult.testFilePath || "unknown";
    const suiteName = filePath.split("/").pop();

    msg("testSuiteStarted", { name: suiteName });

    for (const tc of testResult.testResults) {
      const name =
        tc.ancestorTitles.length > 0
          ? tc.ancestorTitles.join(" > ") + " > " + tc.title
          : tc.title;

      msg("testStarted", { name });

      if (tc.status === "failed") {
        const message = tc.failureMessages.join("\n") || "Test failed";
        msg("testFailed", { name, message, details: message });
      } else if (
        tc.status === "pending" ||
        tc.status === "skipped" ||
        tc.status === "todo" ||
        tc.status === "disabled"
      ) {
        msg("testIgnored", { name, message: tc.status });
      }

      const duration = tc.duration || 0;
      msg("testFinished", { name, duration });
    }

    msg("testSuiteFinished", { name: suiteName });
  }
}

module.exports = LazyTestJestReporter;
