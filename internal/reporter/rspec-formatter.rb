# LazyTest custom RSpec formatter - TeamCity streaming output.
# Emits TeamCity service messages per example for real-time parsing.
# Compatible with RSpec 3+.
#
# Usage: rspec --require /path/to/this/file --format LazyTestTeamCityFormatter

$stdout.sync = true

class LazyTestTeamCityFormatter
  RSpec::Core::Formatters.register self,
    :example_group_started,
    :example_group_finished,
    :example_started,
    :example_passed,
    :example_failed,
    :example_pending

  def initialize(output)
    @output = output
  end

  def example_group_started(notification)
    name = notification.group.description
    @output.puts "##teamcity[testSuiteStarted name='#{tc_escape(name)}']"
  end

  def example_group_finished(notification)
    name = notification.group.description
    @output.puts "##teamcity[testSuiteFinished name='#{tc_escape(name)}']"
  end

  def example_started(notification)
    name = notification.example.description
    @output.puts "##teamcity[testStarted name='#{tc_escape(name)}']"
  end

  def example_passed(notification)
    name = notification.example.description
    duration = (notification.example.execution_result.run_time * 1000).round
    @output.puts "##teamcity[testFinished name='#{tc_escape(name)}' duration='#{duration}']"
  end

  def example_failed(notification)
    name = notification.example.description
    exception = notification.example.execution_result.exception
    message = exception ? exception.message : "Test failed"
    details = exception ? exception.backtrace&.join("\n") || "" : ""
    duration = (notification.example.execution_result.run_time * 1000).round
    @output.puts "##teamcity[testFailed name='#{tc_escape(name)}' message='#{tc_escape(message)}' details='#{tc_escape(details)}']"
    @output.puts "##teamcity[testFinished name='#{tc_escape(name)}' duration='#{duration}']"
  end

  def example_pending(notification)
    name = notification.example.description
    message = notification.example.execution_result.pending_message || "pending"
    @output.puts "##teamcity[testIgnored name='#{tc_escape(name)}' message='#{tc_escape(message)}']"
    @output.puts "##teamcity[testFinished name='#{tc_escape(name)}' duration='0']"
  end

  private

  def tc_escape(str)
    return "" if str.nil?
    str.to_s
      .gsub("|", "||")
      .gsub("'", "|'")
      .gsub("\n", "|n")
      .gsub("\r", "|r")
      .gsub("[", "|[")
      .gsub("]", "|]")
  end
end
