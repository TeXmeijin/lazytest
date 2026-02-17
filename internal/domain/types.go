package domain

import "time"

// TestStatus represents the status of a test case.
type TestStatus int

const (
	StatusPending TestStatus = iota
	StatusRunning
	StatusPassed
	StatusFailed
	StatusSkipped
)

func (s TestStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusPassed:
		return "passed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

func (s TestStatus) Icon() string {
	switch s {
	case StatusPending:
		return "○"
	case StatusRunning:
		return "◉"
	case StatusPassed:
		return "✓"
	case StatusFailed:
		return "✗"
	case StatusSkipped:
		return "⊘"
	default:
		return "?"
	}
}

// TestCase represents a single test method.
type TestCase struct {
	Name     string
	Suite    string
	Status   TestStatus
	Duration time.Duration
	Message  string // failure message
	Details  string // stack trace or additional details
}

// TestSuite represents a group of test cases (typically one test class).
type TestSuite struct {
	Name     string
	Tests    []*TestCase
	Status   TestStatus
	Duration time.Duration
}

// TestFile represents a test file with its previous run status.
type TestFile struct {
	Path       string     // relative path
	PrevStatus TestStatus // status from previous run
}

// TestRun represents the results of a single test execution.
type TestRun struct {
	Suites   []*TestSuite
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Files    []string // files that were tested
}

// SuiteStatus computes the aggregate status for a suite.
func (s *TestSuite) ComputeStatus() TestStatus {
	hasRunning := false
	hasFailed := false
	hasPending := false
	for _, tc := range s.Tests {
		switch tc.Status {
		case StatusFailed:
			hasFailed = true
		case StatusRunning:
			hasRunning = true
		case StatusPending:
			hasPending = true
		}
	}
	if hasFailed {
		return StatusFailed
	}
	if hasRunning {
		return StatusRunning
	}
	if hasPending {
		return StatusPending
	}
	return StatusPassed
}
