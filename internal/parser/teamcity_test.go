package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/meijin/lazytest/internal/domain"
)

func TestParseSuiteStarted(t *testing.T) {
	ev := ParseLine("##teamcity[testSuiteStarted name='Tests\\Feature\\LoginTest']")
	if ev == nil {
		t.Fatal("expected event, got nil")
	}
	if ev.Type != EventSuiteStarted {
		t.Errorf("Type = %d, want EventSuiteStarted", ev.Type)
	}
	if ev.Name != `Tests\Feature\LoginTest` {
		t.Errorf("Name = %q", ev.Name)
	}
}

func TestParseTestStarted(t *testing.T) {
	ev := ParseLine("##teamcity[testStarted name='test_login_succeeds' captureStandardOutput='true']")
	if ev == nil {
		t.Fatal("expected event, got nil")
	}
	if ev.Type != EventTestStarted {
		t.Errorf("Type = %d, want EventTestStarted", ev.Type)
	}
	if ev.Name != "test_login_succeeds" {
		t.Errorf("Name = %q", ev.Name)
	}
}

func TestParseTestFinished(t *testing.T) {
	ev := ParseLine("##teamcity[testFinished name='test_login_succeeds' duration='12']")
	if ev == nil {
		t.Fatal("expected event, got nil")
	}
	if ev.Type != EventTestFinished {
		t.Errorf("Type = %d, want EventTestFinished", ev.Type)
	}
	if ev.Duration != 12*time.Millisecond {
		t.Errorf("Duration = %v", ev.Duration)
	}
}

func TestParseTestFailed(t *testing.T) {
	ev := ParseLine("##teamcity[testFailed name='test_login_fails' message='Expected 401 but got 200' details='at LoginTest.php:42']")
	if ev == nil {
		t.Fatal("expected event, got nil")
	}
	if ev.Type != EventTestFailed {
		t.Errorf("Type = %d, want EventTestFailed", ev.Type)
	}
	if ev.Message != "Expected 401 but got 200" {
		t.Errorf("Message = %q", ev.Message)
	}
	if ev.Details != "at LoginTest.php:42" {
		t.Errorf("Details = %q", ev.Details)
	}
}

func TestParseTestIgnored(t *testing.T) {
	ev := ParseLine("##teamcity[testIgnored name='test_skip' message='Not implemented']")
	if ev == nil {
		t.Fatal("expected event, got nil")
	}
	if ev.Type != EventTestIgnored {
		t.Errorf("Type = %d, want EventTestIgnored", ev.Type)
	}
	if ev.Message != "Not implemented" {
		t.Errorf("Message = %q", ev.Message)
	}
}

func TestEscapeSequences(t *testing.T) {
	ev := ParseLine("##teamcity[testFailed name='test' message='it|'s a |[test|]' details='line1|nline2||pipe']")
	if ev == nil {
		t.Fatal("expected event, got nil")
	}
	if ev.Message != "it's a [test]" {
		t.Errorf("Message = %q, want %q", ev.Message, "it's a [test]")
	}
	if ev.Details != "line1\nline2|pipe" {
		t.Errorf("Details = %q, want %q", ev.Details, "line1\nline2|pipe")
	}
}

func TestNonTeamCityLineReturnsNil(t *testing.T) {
	ev := ParseLine("PHPUnit 10.0.0 by Sebastian Bergmann")
	if ev != nil {
		t.Errorf("expected nil, got event type %d", ev.Type)
	}
}

func TestParseStream(t *testing.T) {
	input := `PHPUnit 10.0
##teamcity[testSuiteStarted name='LoginTest']
##teamcity[testStarted name='test_ok' captureStandardOutput='true']
##teamcity[testFinished name='test_ok' duration='5']
##teamcity[testStarted name='test_fail' captureStandardOutput='true']
##teamcity[testFailed name='test_fail' message='assertion failed' details='trace']
##teamcity[testFinished name='test_fail' duration='3']
##teamcity[testSuiteFinished name='LoginTest']
`
	events := make(chan *Event, 100)
	go ParseStream(strings.NewReader(input), events)

	var collected []*Event
	for ev := range events {
		collected = append(collected, ev)
	}

	// PHPUnit line + 6 teamcity events + empty line
	if len(collected) < 7 {
		t.Fatalf("got %d events, want at least 7", len(collected))
	}

	// First event should be output (PHPUnit line)
	if collected[0].Type != EventOutput {
		t.Errorf("first event type = %d, want EventOutput", collected[0].Type)
	}
}

func TestBuildTestRun(t *testing.T) {
	events := []*Event{
		{Type: EventSuiteStarted, Name: "LoginTest"},
		{Type: EventTestStarted, Name: "test_ok"},
		{Type: EventTestFinished, Name: "test_ok", Duration: 5 * time.Millisecond},
		{Type: EventTestStarted, Name: "test_fail"},
		{Type: EventTestFailed, Name: "test_fail", Message: "failed"},
		{Type: EventTestFinished, Name: "test_fail", Duration: 3 * time.Millisecond},
		{Type: EventTestStarted, Name: "test_skip"},
		{Type: EventTestIgnored, Name: "test_skip"},
		{Type: EventTestFinished, Name: "test_skip", Duration: 1 * time.Millisecond},
		{Type: EventSuiteFinished, Name: "LoginTest"},
	}

	run := BuildTestRun(events)

	if run.Passed != 1 {
		t.Errorf("Passed = %d, want 1", run.Passed)
	}
	if run.Failed != 1 {
		t.Errorf("Failed = %d, want 1", run.Failed)
	}
	if run.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", run.Skipped)
	}
	if len(run.Suites) != 1 {
		t.Fatalf("Suites = %d, want 1", len(run.Suites))
	}
	if run.Suites[0].Status != domain.StatusFailed {
		t.Errorf("Suite status = %v, want Failed", run.Suites[0].Status)
	}
}
