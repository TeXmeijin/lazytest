package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/meijin/lazytest/internal/domain"
)

// matchedFile holds a file along with its fuzzy match metadata.
type matchedFile struct {
	file    domain.TestFile
	score   int
	indices []int // byte indices in file.Path that matched the query
}

// fuzzyScore returns whether query fuzzy-matches str, along with a score and
// the matched byte indices. Higher score = better match.
//
// Two-phase approach inspired by fzf:
//  1. Try contiguous substring match first (highest quality).
//  2. Fall back to fuzzy subsequence with backward pass for tightest window,
//     requiring at least one consecutive character pair for queries of 3+ chars.
func fuzzyScore(query, str string) (bool, int, []int) {
	if query == "" {
		return true, 0, nil
	}

	lower := strings.ToLower(str)
	queryLower := strings.ToLower(query)
	qLen := len(queryLower)

	// Phase 1: Contiguous substring match — always preferred.
	if idx := strings.Index(lower, queryLower); idx != -1 {
		indices := make([]int, qLen)
		for i := 0; i < qLen; i++ {
			indices[i] = idx + i
		}
		score := 100 + (qLen-1)*15 // big base + full consecutive bonus
		score += boundaryBonus(lower, idx)
		score -= len(str) / 4
		return true, score, indices
	}

	// Phase 2: Fuzzy subsequence match.

	// Forward pass: verify match exists and find the earliest end position.
	qi := 0
	endPos := -1
	for si := 0; si < len(lower); si++ {
		if lower[si] == queryLower[qi] {
			qi++
			if qi == qLen {
				endPos = si
				break
			}
		}
	}
	if qi < qLen {
		return false, 0, nil
	}

	// Backward pass: from endPos, scan backward to find the tightest window.
	indices := make([]int, qLen)
	qi = qLen - 1
	for si := endPos; si >= 0 && qi >= 0; si-- {
		if lower[si] == queryLower[qi] {
			indices[qi] = si
			qi--
		}
	}

	// Count consecutive pairs.
	consecutive := 0
	for i := 1; i < len(indices); i++ {
		if indices[i] == indices[i-1]+1 {
			consecutive++
		}
	}

	// For queries of 3+ chars, require at least one consecutive pair.
	// This filters out completely scattered matches like t...u...t...o...r.
	if qLen >= 3 && consecutive == 0 {
		return false, 0, nil
	}

	// Hard cutoff on span.
	span := indices[qLen-1] - indices[0] + 1
	if span > qLen*4 {
		return false, 0, nil
	}

	// Scoring.
	score := consecutive * 15
	for _, idx := range indices {
		score += boundaryBonus(lower, idx)
	}
	overrun := span - qLen
	score -= overrun * 3
	score -= len(str) / 4

	return true, score, indices
}

// boundaryBonus returns a bonus if pos is at a word boundary in str.
func boundaryBonus(str string, pos int) int {
	if pos == 0 {
		return 10
	}
	prev := str[pos-1]
	if prev == '/' || prev == '_' || prev == '-' || prev == '.' {
		return 10
	}
	return 0
}

// filterFuzzy filters files by fuzzy match and sorts best matches first.
// When query is empty, all files are returned in original order.
func filterFuzzy(files []domain.TestFile, query string) []matchedFile {
	if query == "" {
		result := make([]matchedFile, len(files))
		for i, f := range files {
			result[i] = matchedFile{file: f}
		}
		return result
	}

	var result []matchedFile
	for _, f := range files {
		ok, score, indices := fuzzyScore(query, f.Path)
		if ok {
			result = append(result, matchedFile{file: f, score: score, indices: indices})
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].score > result[j].score
	})

	return result
}

// renderWithHighlight renders path with matched characters styled differently.
func renderWithHighlight(path string, indices []int, base, highlight lipgloss.Style) string {
	if len(indices) == 0 {
		return base.Render(path)
	}

	indexSet := make(map[int]bool, len(indices))
	for _, i := range indices {
		indexSet[i] = true
	}

	var sb strings.Builder
	i := 0
	for i < len(path) {
		if indexSet[i] {
			// Collect a run of matched characters.
			j := i
			for j < len(path) && indexSet[j] {
				j++
			}
			sb.WriteString(highlight.Render(path[i:j]))
			i = j
		} else {
			// Collect a run of unmatched characters.
			j := i
			for j < len(path) && !indexSet[j] {
				j++
			}
			sb.WriteString(base.Render(path[i:j]))
			i = j
		}
	}
	return sb.String()
}
