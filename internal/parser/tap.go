package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	tapResultRe   = regexp.MustCompile(`^(not ok|ok)\s+(\d+)\s+-\s+(.+)$`)
	tapDurationRe = regexp.MustCompile(`time=([0-9.]+)ms`)
	tapSkipRe     = regexp.MustCompile(`(?i)^SKIP\b(.*)`)
	tapPlanRe     = regexp.MustCompile(`^1\.\.\d+$`)
)

// TAPParser parses TAP v13 flat format output into Events.
type TAPParser struct {
	events       chan<- *Event
	currentSuite string
	inYAMLBlock  bool
	yamlLines    []string
	pendingTest  *pendingTAPTest
}

type pendingTAPTest struct {
	name     string
	duration time.Duration
}

// NewTAPParser creates a new TAP parser that sends events to the given channel.
func NewTAPParser(events chan<- *Event) *TAPParser {
	return &TAPParser{events: events}
}

// ParseLine processes a single line of TAP output.
func (p *TAPParser) ParseLine(line string) {
	trimmed := strings.TrimSpace(line)

	// Handle YAML block state
	if p.inYAMLBlock {
		if trimmed == "..." {
			p.inYAMLBlock = false
			p.emitPendingWithYAML()
			return
		}
		p.yamlLines = append(p.yamlLines, trimmed)
		return
	}

	// Check for YAML block start (only after a "not ok" line)
	if p.pendingTest != nil && trimmed == "---" {
		p.inYAMLBlock = true
		p.yamlLines = nil
		return
	}

	// If we have a pending test and this is NOT a YAML start, emit it without YAML
	if p.pendingTest != nil {
		p.emitPendingWithoutYAML()
	}

	// Skip TAP version and plan lines
	if strings.HasPrefix(trimmed, "TAP version") {
		return
	}
	if tapPlanRe.MatchString(trimmed) {
		return
	}

	// Try to match a test result line
	m := tapResultRe.FindStringSubmatch(trimmed)
	if m == nil {
		if trimmed != "" {
			p.events <- &Event{Type: EventOutput, RawLine: line}
		}
		return
	}

	status := m[1]
	fullNameAndDirective := m[3]

	// Split directive from name using last occurrence of " # "
	var fullName, directive string
	if idx := strings.LastIndex(fullNameAndDirective, " # "); idx >= 0 {
		fullName = fullNameAndDirective[:idx]
		directive = fullNameAndDirective[idx+3:]
	} else {
		fullName = fullNameAndDirective
	}

	suite, testName := splitTAPName(fullName)

	// Handle suite transition
	if suite != p.currentSuite {
		if p.currentSuite != "" {
			p.events <- &Event{Type: EventSuiteFinished, Name: p.currentSuite}
		}
		p.currentSuite = suite
		p.events <- &Event{Type: EventSuiteStarted, Name: suite}
	}

	// Parse duration from directive
	var dur time.Duration
	if directive != "" {
		if dm := tapDurationRe.FindStringSubmatch(directive); dm != nil {
			if fms, err := strconv.ParseFloat(dm[1], 64); err == nil {
				dur = time.Duration(fms * float64(time.Millisecond))
			}
		}
	}

	if status == "not ok" {
		p.pendingTest = &pendingTAPTest{
			name:     testName,
			duration: dur,
		}
		return
	}

	// "ok" line - check for SKIP directive
	if directive != "" {
		if sm := tapSkipRe.FindStringSubmatch(directive); sm != nil {
			reason := strings.TrimSpace(sm[1])
			p.events <- &Event{Type: EventTestStarted, Name: testName}
			p.events <- &Event{Type: EventTestIgnored, Name: testName, Message: reason}
			p.events <- &Event{Type: EventTestFinished, Name: testName, Duration: dur}
			return
		}
	}

	// Passing test
	p.events <- &Event{Type: EventTestStarted, Name: testName}
	p.events <- &Event{Type: EventTestFinished, Name: testName, Duration: dur}
}

func (p *TAPParser) emitPendingWithYAML() {
	if p.pendingTest == nil {
		return
	}
	pt := p.pendingTest
	p.pendingTest = nil

	message, details := parseYAMLBlock(p.yamlLines)

	p.events <- &Event{Type: EventTestStarted, Name: pt.name}
	p.events <- &Event{Type: EventTestFailed, Name: pt.name, Message: message, Details: details}
	p.events <- &Event{Type: EventTestFinished, Name: pt.name, Duration: pt.duration}
}

func (p *TAPParser) emitPendingWithoutYAML() {
	if p.pendingTest == nil {
		return
	}
	pt := p.pendingTest
	p.pendingTest = nil

	p.events <- &Event{Type: EventTestStarted, Name: pt.name}
	p.events <- &Event{Type: EventTestFailed, Name: pt.name}
	p.events <- &Event{Type: EventTestFinished, Name: pt.name, Duration: pt.duration}
}

// Flush finalizes the parser, emitting any pending events and closing the last suite.
func (p *TAPParser) Flush() {
	if p.inYAMLBlock && p.pendingTest != nil {
		p.inYAMLBlock = false
		p.emitPendingWithYAML()
	} else if p.pendingTest != nil {
		p.emitPendingWithoutYAML()
	}
	if p.currentSuite != "" {
		p.events <- &Event{Type: EventSuiteFinished, Name: p.currentSuite}
		p.currentSuite = ""
	}
}

// splitTAPName splits a TAP flat test name (separated by " > ") into suite and test name.
func splitTAPName(fullName string) (suite, testName string) {
	parts := strings.Split(fullName, " > ")
	switch len(parts) {
	case 1:
		return parts[0], parts[0]
	case 2:
		return parts[0], parts[1]
	default:
		// 3+ parts: parts[0] is file, parts[1:n-1] are suite hierarchy, parts[n-1] is test name
		suite = strings.Join(parts[1:len(parts)-1], " > ")
		testName = parts[len(parts)-1]
		return suite, testName
	}
}

// parseYAMLBlock extracts message and details from YAML-like lines between --- and ...
func parseYAMLBlock(lines []string) (message, details string) {
	var actual, expected, at, msg string
	for _, line := range lines {
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		val = strings.Trim(val, "\"'")
		switch key {
		case "message":
			msg = val
		case "at":
			at = val
		case "actual":
			actual = val
		case "expected":
			expected = val
		}
	}

	if msg != "" {
		message = msg
	} else if actual != "" || expected != "" {
		message = "expected: " + expected + ", actual: " + actual
	}
	if at != "" {
		details = at
	}
	return message, details
}
