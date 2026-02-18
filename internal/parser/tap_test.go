package parser

import (
	"strings"
	"testing"
	"time"
)

func collectTAPEvents(input string) []*Event {
	events := make(chan *Event, 100)
	tap := NewTAPParser(events)
	for _, line := range strings.Split(input, "\n") {
		tap.ParseLine(line)
	}
	tap.Flush()
	close(events)

	var collected []*Event
	for ev := range events {
		collected = append(collected, ev)
	}
	return collected
}

func TestTAPPassingTest(t *testing.T) {
	input := `ok 1 - src/App.test.ts > AppComponent > renders correctly # time=5.23ms`
	events := collectTAPEvents(input)

	// SuiteStarted + TestStarted + TestFinished + SuiteFinished(flush)
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4", len(events))
	}
	if events[0].Type != EventSuiteStarted || events[0].Name != "AppComponent" {
		t.Errorf("event[0] = %+v, want SuiteStarted 'AppComponent'", events[0])
	}
	if events[1].Type != EventTestStarted || events[1].Name != "renders correctly" {
		t.Errorf("event[1] = %+v, want TestStarted 'renders correctly'", events[1])
	}
	if events[2].Type != EventTestFinished || events[2].Name != "renders correctly" {
		t.Errorf("event[2] = %+v, want TestFinished 'renders correctly'", events[2])
	}
	expected := time.Duration(5.23 * float64(time.Millisecond))
	if events[2].Duration != expected {
		t.Errorf("duration = %v, want %v", events[2].Duration, expected)
	}
	if events[3].Type != EventSuiteFinished || events[3].Name != "AppComponent" {
		t.Errorf("event[3] = %+v, want SuiteFinished 'AppComponent'", events[3])
	}
}

func TestTAPFailingTestWithYAML(t *testing.T) {
	input := `not ok 1 - src/App.test.ts > AppComponent > validates input # time=10.5ms
  ---
  message: "expected true to be false"
  at: "src/App.test.ts:42:5"
  actual: "true"
  expected: "false"
  ...`
	events := collectTAPEvents(input)

	// SuiteStarted + TestStarted + TestFailed + TestFinished + SuiteFinished(flush)
	if len(events) != 5 {
		t.Fatalf("got %d events, want 5; events: %+v", len(events), events)
	}
	if events[0].Type != EventSuiteStarted {
		t.Errorf("event[0].Type = %d, want SuiteStarted", events[0].Type)
	}
	if events[1].Type != EventTestStarted || events[1].Name != "validates input" {
		t.Errorf("event[1] = %+v", events[1])
	}
	if events[2].Type != EventTestFailed {
		t.Errorf("event[2].Type = %d, want TestFailed", events[2].Type)
	}
	if events[2].Message != "expected true to be false" {
		t.Errorf("message = %q", events[2].Message)
	}
	if events[2].Details != "src/App.test.ts:42:5" {
		t.Errorf("details = %q", events[2].Details)
	}
	if events[3].Type != EventTestFinished {
		t.Errorf("event[3].Type = %d, want TestFinished", events[3].Type)
	}
}

func TestTAPSkippedTest(t *testing.T) {
	input := `ok 1 - src/App.test.ts > AppComponent > todo test # SKIP not implemented`
	events := collectTAPEvents(input)

	// SuiteStarted + TestStarted + TestIgnored + TestFinished + SuiteFinished(flush)
	if len(events) != 5 {
		t.Fatalf("got %d events, want 5", len(events))
	}
	if events[2].Type != EventTestIgnored {
		t.Errorf("event[2].Type = %d, want TestIgnored", events[2].Type)
	}
	if events[2].Message != "not implemented" {
		t.Errorf("message = %q, want 'not implemented'", events[2].Message)
	}
}

func TestTAPSuiteTransition(t *testing.T) {
	input := `ok 1 - src/App.test.ts > SuiteA > test1 # time=1ms
ok 2 - src/utils.test.ts > SuiteB > test2 # time=2ms`
	events := collectTAPEvents(input)

	expected := []EventType{
		EventSuiteStarted,  // SuiteA
		EventTestStarted,   // test1
		EventTestFinished,  // test1
		EventSuiteFinished, // SuiteA
		EventSuiteStarted,  // SuiteB
		EventTestStarted,   // test2
		EventTestFinished,  // test2
		EventSuiteFinished, // SuiteB (from Flush)
	}

	if len(events) != len(expected) {
		types := make([]EventType, len(events))
		for i, e := range events {
			types[i] = e.Type
		}
		t.Fatalf("got %d events %v, want %d %v", len(events), types, len(expected), expected)
	}
	for i, typ := range expected {
		if events[i].Type != typ {
			t.Errorf("event[%d].Type = %d, want %d", i, events[i].Type, typ)
		}
	}
}

func TestTAPSameSuiteContinuous(t *testing.T) {
	input := `ok 1 - src/App.test.ts > SuiteA > test1 # time=1ms
ok 2 - src/App.test.ts > SuiteA > test2 # time=2ms`
	events := collectTAPEvents(input)

	suitesStarted := 0
	for _, e := range events {
		if e.Type == EventSuiteStarted {
			suitesStarted++
		}
	}
	if suitesStarted != 1 {
		t.Errorf("SuiteStarted count = %d, want 1", suitesStarted)
	}
}

func TestTAPFailingWithoutYAML(t *testing.T) {
	input := `not ok 1 - src/App.test.ts > SuiteA > test1
ok 2 - src/App.test.ts > SuiteA > test2`
	events := collectTAPEvents(input)

	hasTestFailed := false
	for _, e := range events {
		if e.Type == EventTestFailed && e.Name == "test1" {
			hasTestFailed = true
		}
	}
	if !hasTestFailed {
		t.Error("expected TestFailed event for test1")
	}
}

func TestTAPDurationParse(t *testing.T) {
	input := `ok 1 - src/App.test.ts > SuiteA > test1 # time=123.45ms`
	events := collectTAPEvents(input)

	var dur time.Duration
	for _, e := range events {
		if e.Type == EventTestFinished {
			dur = e.Duration
		}
	}
	expected := time.Duration(123.45 * float64(time.Millisecond))
	if dur != expected {
		t.Errorf("duration = %v, want %v", dur, expected)
	}
}

func TestTAPTwoPartName(t *testing.T) {
	input := `ok 1 - myFile.test.ts > myTest # time=1ms`
	events := collectTAPEvents(input)

	if events[0].Type != EventSuiteStarted || events[0].Name != "myFile.test.ts" {
		t.Errorf("suite = %q, want 'myFile.test.ts'", events[0].Name)
	}
	if events[1].Type != EventTestStarted || events[1].Name != "myTest" {
		t.Errorf("test = %q, want 'myTest'", events[1].Name)
	}
}

func TestTAPDeepNesting(t *testing.T) {
	input := `ok 1 - src/App.test.ts > outer > inner > deepTest # time=1ms`
	events := collectTAPEvents(input)

	if events[0].Type != EventSuiteStarted || events[0].Name != "outer > inner" {
		t.Errorf("suite = %q, want 'outer > inner'", events[0].Name)
	}
	if events[1].Name != "deepTest" {
		t.Errorf("test = %q, want 'deepTest'", events[1].Name)
	}
}

func TestTAPFlushClosesSuite(t *testing.T) {
	input := `ok 1 - src/App.test.ts > SuiteA > test1`
	events := collectTAPEvents(input)

	last := events[len(events)-1]
	if last.Type != EventSuiteFinished || last.Name != "SuiteA" {
		t.Errorf("last event = %+v, want SuiteFinished 'SuiteA'", last)
	}
}

func TestTAPVersionAndPlanSkipped(t *testing.T) {
	input := `TAP version 13
1..3`
	events := collectTAPEvents(input)

	if len(events) != 0 {
		t.Errorf("got %d events, want 0 (version and plan lines should be skipped)", len(events))
	}
}

func TestTAPFullStreamE2E(t *testing.T) {
	input := `TAP version 13
1..5
ok 1 - src/App.test.ts > AppComponent > renders correctly # time=5.23ms
ok 2 - src/App.test.ts > AppComponent > handles click # time=3.12ms
not ok 3 - src/App.test.ts > AppComponent > validates input # time=10.5ms
  ---
  message: "expected true to be false"
  at: "src/App.test.ts:42:5"
  ...
ok 4 - src/utils.test.ts > formatDate > formats correctly # time=1.2ms
ok 5 - src/utils.test.ts > formatDate > handles null # SKIP not implemented`

	// Use ParseStream for auto-detection
	events := make(chan *Event, 100)
	go ParseStream(strings.NewReader(input), events)

	var collected []*Event
	for ev := range events {
		collected = append(collected, ev)
	}

	suiteStarts := 0
	suiteFinishes := 0
	testStarts := 0
	testFinishes := 0
	testFails := 0
	testIgnores := 0
	for _, e := range collected {
		switch e.Type {
		case EventSuiteStarted:
			suiteStarts++
		case EventSuiteFinished:
			suiteFinishes++
		case EventTestStarted:
			testStarts++
		case EventTestFinished:
			testFinishes++
		case EventTestFailed:
			testFails++
		case EventTestIgnored:
			testIgnores++
		}
	}

	if suiteStarts != 2 {
		t.Errorf("SuiteStarted = %d, want 2", suiteStarts)
	}
	if suiteFinishes != 2 {
		t.Errorf("SuiteFinished = %d, want 2", suiteFinishes)
	}
	if testStarts != 5 {
		t.Errorf("TestStarted = %d, want 5", testStarts)
	}
	if testFinishes != 5 {
		t.Errorf("TestFinished = %d, want 5", testFinishes)
	}
	if testFails != 1 {
		t.Errorf("TestFailed = %d, want 1", testFails)
	}
	if testIgnores != 1 {
		t.Errorf("TestIgnored = %d, want 1", testIgnores)
	}

	// Verify BuildTestRun compatibility
	run := BuildTestRun(collected)
	if run.Passed != 3 {
		t.Errorf("Passed = %d, want 3", run.Passed)
	}
	if run.Failed != 1 {
		t.Errorf("Failed = %d, want 1", run.Failed)
	}
	if run.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", run.Skipped)
	}
}
