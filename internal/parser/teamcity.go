package parser

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/meijin/lazytest/internal/domain"
)

// EventType represents the type of a TeamCity event.
type EventType int

const (
	EventSuiteStarted EventType = iota
	EventSuiteFinished
	EventTestStarted
	EventTestFinished
	EventTestFailed
	EventTestIgnored
	EventOutput
)

// Event represents a parsed TeamCity event.
type Event struct {
	Type     EventType
	Name     string
	Duration time.Duration
	Message  string
	Details  string
	RawLine  string
}

// ParseLine parses a single line of TeamCity output.
// Returns nil if the line is not a TeamCity message.
func ParseLine(line string) *Event {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "##teamcity[") || !strings.HasSuffix(line, "]") {
		return nil
	}

	// Extract inner content: between "##teamcity[" and "]"
	inner := line[len("##teamcity[") : len(line)-1]

	// Split message name from attributes
	spaceIdx := strings.IndexByte(inner, ' ')
	if spaceIdx < 0 {
		return nil
	}

	msgType := inner[:spaceIdx]
	attrs := parseAttributes(inner[spaceIdx+1:])

	name := attrs["name"]

	switch msgType {
	case "testSuiteStarted":
		return &Event{Type: EventSuiteStarted, Name: name, RawLine: line}
	case "testSuiteFinished":
		return &Event{Type: EventSuiteFinished, Name: name, RawLine: line}
	case "testStarted":
		return &Event{Type: EventTestStarted, Name: name, RawLine: line}
	case "testFinished":
		dur := parseDuration(attrs["duration"])
		return &Event{Type: EventTestFinished, Name: name, Duration: dur, RawLine: line}
	case "testFailed":
		return &Event{
			Type:    EventTestFailed,
			Name:    name,
			Message: attrs["message"],
			Details: attrs["details"],
			RawLine: line,
		}
	case "testIgnored":
		return &Event{
			Type:    EventTestIgnored,
			Name:    name,
			Message: attrs["message"],
			RawLine: line,
		}
	default:
		return nil
	}
}

// parseAttributes parses key='value' pairs from a TeamCity message.
func parseAttributes(s string) map[string]string {
	attrs := make(map[string]string)

	for len(s) > 0 {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			break
		}

		// Find key
		eqIdx := strings.IndexByte(s, '=')
		if eqIdx < 0 {
			break
		}
		key := s[:eqIdx]
		s = s[eqIdx+1:]

		// Value must start with '
		if len(s) == 0 || s[0] != '\'' {
			break
		}
		s = s[1:] // skip opening quote

		// Find closing quote (not escaped)
		var value strings.Builder
		i := 0
		for i < len(s) {
			if s[i] == '\'' {
				// End of value
				s = s[i+1:]
				break
			}
			if s[i] == '|' && i+1 < len(s) {
				// Escape sequence
				next := s[i+1]
				switch next {
				case '\'':
					value.WriteByte('\'')
				case 'n':
					value.WriteByte('\n')
				case 'r':
					value.WriteByte('\r')
				case '|':
					value.WriteByte('|')
				case '[':
					value.WriteByte('[')
				case ']':
					value.WriteByte(']')
				default:
					value.WriteByte('|')
					value.WriteByte(next)
				}
				i += 2
				continue
			}
			value.WriteByte(s[i])
			i++
		}

		attrs[key] = value.String()
	}

	return attrs
}

func parseDuration(s string) time.Duration {
	// Try integer first (PHPUnit outputs integer milliseconds)
	ms, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return time.Duration(ms) * time.Millisecond
	}
	// Try float (Vitest outputs fractional milliseconds)
	fms, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return time.Duration(fms * float64(time.Millisecond))
	}
	return 0
}

// ParseStream reads from an io.Reader line by line and sends Events to the channel.
func ParseStream(r io.Reader, events chan<- *Event) {
	defer close(events)
	scanner := bufio.NewScanner(r)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		ev := ParseLine(line)
		if ev == nil {
			// Non-TeamCity output
			events <- &Event{Type: EventOutput, RawLine: line}
		} else {
			events <- ev
		}
	}
}

// BuildTestRun processes a slice of events and builds a TestRun result.
func BuildTestRun(events []*Event) *domain.TestRun {
	run := &domain.TestRun{}
	var suiteMap = make(map[string]*domain.TestSuite)
	var currentSuite *domain.TestSuite

	for _, ev := range events {
		switch ev.Type {
		case EventSuiteStarted:
			suite := &domain.TestSuite{
				Name:   ev.Name,
				Status: domain.StatusRunning,
			}
			suiteMap[ev.Name] = suite
			run.Suites = append(run.Suites, suite)
			currentSuite = suite

		case EventSuiteFinished:
			if s, ok := suiteMap[ev.Name]; ok {
				s.Status = s.ComputeStatus()
			}

		case EventTestStarted:
			if currentSuite != nil {
				tc := &domain.TestCase{
					Name:   ev.Name,
					Suite:  currentSuite.Name,
					Status: domain.StatusRunning,
				}
				currentSuite.Tests = append(currentSuite.Tests, tc)
			}

		case EventTestFinished:
			if currentSuite != nil {
				for _, tc := range currentSuite.Tests {
					if tc.Name == ev.Name {
						if tc.Status == domain.StatusRunning {
							tc.Status = domain.StatusPassed
						}
						tc.Duration = ev.Duration
						run.Duration += ev.Duration
						break
					}
				}
			}

		case EventTestFailed:
			if currentSuite != nil {
				for _, tc := range currentSuite.Tests {
					if tc.Name == ev.Name {
						tc.Status = domain.StatusFailed
						tc.Message = ev.Message
						tc.Details = ev.Details
						break
					}
				}
			}

		case EventTestIgnored:
			if currentSuite != nil {
				for _, tc := range currentSuite.Tests {
					if tc.Name == ev.Name {
						tc.Status = domain.StatusSkipped
						tc.Message = ev.Message
						break
					}
				}
			}
		}
	}

	// Count results
	for _, suite := range run.Suites {
		for _, tc := range suite.Tests {
			switch tc.Status {
			case domain.StatusPassed:
				run.Passed++
			case domain.StatusFailed:
				run.Failed++
			case domain.StatusSkipped:
				run.Skipped++
			}
		}
	}

	return run
}
