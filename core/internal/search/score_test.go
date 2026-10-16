package search

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func rank(query string, fuzzy bool, rels ...string) []string {
	terms := parseTerms(query)
	type scored struct {
		rel   string
		score int
	}
	var out []scored
	for _, rel := range rels {
		name := rel
		depth := 0
		for i := len(rel) - 1; i >= 0; i-- {
			if rel[i] == '/' {
				name = rel[i+1:]
				break
			}
		}
		for _, c := range rel {
			if c == '/' {
				depth++
			}
		}
		s, tier := score(name, rel, depth, terms, fuzzy)
		if tier == tierNone {
			continue
		}
		out = append(out, scored{rel, s})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	names := make([]string, 0, len(out))
	for _, s := range out {
		names = append(names, s.rel)
	}
	return names
}

func TestScorerOrdersTiers(t *testing.T) {
	got := rank("report", true,
		"docs/notes.txt",
		"r-e-p-o-r-t.txt",
		"reports/summary.md",
		"my-report.pdf",
		"myreport.pdf",
		"reporting.txt",
		"report",
	)
	assert.Equal(t, []string{
		"report",
		"reporting.txt",
		"my-report.pdf",
		"myreport.pdf",
		"reports/summary.md",
		"r-e-p-o-r-t.txt",
	}, got)
}

func TestScorerRequiresEveryTerm(t *testing.T) {
	got := rank("search walk", false, "search/walk.go", "search/service.go", "walk/search_test.go", "notes.txt")
	assert.ElementsMatch(t, []string{"walk/search_test.go", "search/walk.go"}, got)
}

func TestScorerDepthPenaltyBreaksTies(t *testing.T) {
	got := rank("deps", false, "a/b/c/deps.go", "deps.go", "a/deps.go")
	assert.Equal(t, []string{"deps.go", "a/deps.go", "a/b/c/deps.go"}, got)
}

func TestScorerFuzzyIsOptIn(t *testing.T) {
	assert.Empty(t, rank("rpt", false, "report.txt"))
	assert.Equal(t, []string{"report.txt"}, rank("rpt", true, "report.txt"))
}

func TestScorerFuzzyPrefersContiguousRuns(t *testing.T) {
	got := rank("walk", true, "w-a-l-k.txt", "wa-lk.txt", "wal-k.txt")
	assert.Equal(t, []string{"wal-k.txt", "wa-lk.txt", "w-a-l-k.txt"}, got)
}

func TestScorerGlobMatchesWholeName(t *testing.T) {
	assert.Equal(t, []string{"notes.md"}, rank("*.md", false, "notes.md", "md.txt", "readme.markdown"))
	assert.Equal(t, []string{"a1.go"}, rank("a?.go", false, "a1.go", "a12.go"))
}

func TestScorerIsCaseInsensitive(t *testing.T) {
	terms := parseTerms("README")
	s, tier := score("readme.md", "readme.md", 0, terms, false)
	assert.Equal(t, tierPrefix, tier)
	assert.Equal(t, scorePrefix+bonusBoundary, s)
}

func TestScorerEmptyQueryMatchesEverything(t *testing.T) {
	s, tier := score("x", "a/x", 1, nil, false)
	assert.Equal(t, tierSubstring, tier)
	assert.Equal(t, -penaltyDepth, s)
}

func TestWordBoundaryBeatsMidWord(t *testing.T) {
	got := rank("port", false, "import.go", "my_port.go")
	assert.Equal(t, []string{"my_port.go", "import.go"}, got)
}
