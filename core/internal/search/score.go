package search

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	tierNone = iota
	tierFuzzy
	tierPath
	tierSubstring
	tierPrefix
	tierExact
)

const (
	scoreExact     = 1000
	scorePrefix    = 800
	scoreSubstring = 600
	scorePath      = 400
	scoreFuzzy     = 200
	scoreFuzzyPath = 100
	bonusBoundary  = 50
	bonusRun       = 10
	penaltyDepth   = 3
	maxDepthPen    = 60
)

type term struct {
	text string
	glob bool
}

func parseTerms(query string) []term {
	var out []term
	for _, word := range strings.Fields(strings.ToLower(query)) {
		out = append(out, term{text: word, glob: strings.ContainsAny(word, "*?")})
	}
	return out
}

// score ranks an entry whose lowercased name and root-relative path are
// given. The tier is the weakest match among the terms; tierNone means a
// term did not match at all.
func score(nameLower, relLower string, depth int, terms []term, fuzzy bool) (int, int) {
	total := 0
	weakest := tierExact
	parent := relLower[:len(relLower)-len(nameLower)]
	for _, t := range terms {
		points, tier := scoreTerm(nameLower, parent, t, fuzzy)
		if tier == tierNone {
			return 0, tierNone
		}
		total += points
		weakest = min(weakest, tier)
	}
	if len(terms) == 0 {
		weakest = tierSubstring
	}
	return total - min(depth*penaltyDepth, maxDepthPen), weakest
}

func scoreTerm(name, parent string, t term, fuzzy bool) (int, int) {
	switch {
	case t.glob:
		if wildcard(name, t.text) {
			return scoreSubstring + bonusBoundary, tierSubstring
		}
		return 0, tierNone
	case name == t.text:
		return scoreExact + bonusBoundary, tierExact
	case strings.HasPrefix(name, t.text):
		return scorePrefix + bonusBoundary, tierPrefix
	}
	if at, ok := boundaryIndex(name, t.text); ok {
		return scoreSubstring + boundaryBonus(name, at), tierSubstring
	}
	if at, ok := boundaryIndex(parent, t.text); ok {
		return scorePath + boundaryBonus(parent, at), tierPath
	}
	if !fuzzy {
		return 0, tierNone
	}
	if run, start, ok := subsequence(name, t.text); ok {
		return scoreFuzzy + run*bonusRun + boundaryBonus(name, start), tierFuzzy
	}
	if run, start, ok := subsequence(parent, t.text); ok {
		return scoreFuzzyPath + run*bonusRun + boundaryBonus(parent, start), tierFuzzy
	}
	return 0, tierNone
}

// boundaryIndex prefers an occurrence that starts a word over the first one.
func boundaryIndex(hay, needle string) (int, bool) {
	first := strings.Index(hay, needle)
	if first < 0 {
		return 0, false
	}
	for at := first; at >= 0; {
		if atBoundary(hay, at) {
			return at, true
		}
		next := strings.Index(hay[at+1:], needle)
		if next < 0 {
			break
		}
		at += 1 + next
	}
	return first, true
}

func boundaryBonus(hay string, at int) int {
	if atBoundary(hay, at) {
		return bonusBoundary
	}
	return 0
}

func atBoundary(hay string, at int) bool {
	if at == 0 {
		return true
	}
	prev, _ := utf8.DecodeLastRuneInString(hay[:at])
	return !unicode.IsLetter(prev) && !unicode.IsDigit(prev)
}

// subsequence matches needle's characters in order and reports the longest
// contiguous run it found and where the match starts.
func subsequence(hay, needle string) (run, start int, ok bool) {
	if needle == "" {
		return 0, 0, true
	}
	pos := 0
	best, current, last := 0, 0, -2
	for i := range len(needle) {
		next := strings.IndexByte(hay[pos:], needle[i])
		if next < 0 {
			return 0, 0, false
		}
		at := pos + next
		if i == 0 {
			start = at
		}
		switch at {
		case last + 1:
			current++
		default:
			current = 1
		}
		best = max(best, current)
		last = at
		pos = at + 1
	}
	return best - 1, start, true
}

func wildcard(nameText, patternText string) bool {
	name, pattern := []rune(nameText), []rune(patternText)
	at, p := 0, 0
	star, mark := -1, 0
	for at < len(name) {
		switch {
		case p < len(pattern) && (pattern[p] == '?' || pattern[p] == name[at]):
			at++
			p++
		case p < len(pattern) && pattern[p] == '*':
			star = p
			mark = at
			p++
		case star < 0:
			return false
		default:
			p = star + 1
			mark++
			at = mark
		}
	}
	for p < len(pattern) && pattern[p] == '*' {
		p++
	}
	return p == len(pattern)
}
