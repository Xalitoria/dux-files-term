package mimeapps

import (
	"bufio"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

type globEntry struct {
	mime    string
	pattern string
	weight  int
	cs      bool
}

type globSet struct {
	literals   map[string]globEntry
	literalsCS map[string]globEntry
	suffixes   map[string]globEntry
	suffixesCS map[string]globEntry
	others     []globEntry
	maxSuffix  int
}

func newGlobSet() *globSet {
	return &globSet{
		literals:   map[string]globEntry{},
		literalsCS: map[string]globEntry{},
		suffixes:   map[string]globEntry{},
		suffixesCS: map[string]globEntry{},
	}
}

func (g *globSet) load(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if e, ok := parseGlobLine(line); ok {
			g.add(e)
		}
	}
}

func parseGlobLine(line string) (globEntry, bool) {
	fields := strings.SplitN(line, ":", 4)
	if len(fields) < 3 {
		return globEntry{}, false
	}
	weight, err := strconv.Atoi(fields[0])
	if err != nil || fields[1] == "" {
		return globEntry{}, false
	}

	e := globEntry{mime: fields[1], pattern: fields[2], weight: weight}
	// A fourth field is either the flag list or a glob that contained a colon.
	if len(fields) == 4 {
		if isFlagList(fields[3]) {
			e.cs = strings.Contains(fields[3], "cs")
		} else {
			e.pattern += ":" + fields[3]
		}
	}
	if e.pattern == "" {
		return globEntry{}, false
	}
	if !e.cs {
		e.pattern = strings.ToLower(e.pattern)
	}
	return e, true
}

func isFlagList(field string) bool {
	if field == "" {
		return false
	}
	for _, flag := range strings.Split(field, ",") {
		switch flag {
		case "cs", "ci":
		default:
			return false
		}
	}
	return true
}

func (g *globSet) add(e globEntry) {
	literals, suffixes := g.literals, g.suffixes
	if e.cs {
		literals, suffixes = g.literalsCS, g.suffixesCS
	}

	switch {
	case !strings.ContainsAny(e.pattern, "*?["):
		keepBest(literals, e.pattern, e)
	case strings.HasPrefix(e.pattern, "*") && !strings.ContainsAny(e.pattern[1:], "*?["):
		suffix := e.pattern[1:]
		keepBest(suffixes, suffix, e)
		g.maxSuffix = max(g.maxSuffix, len(suffix))
	default:
		g.others = append(g.others, e)
	}
}

func keepBest(index map[string]globEntry, key string, e globEntry) {
	if existing, ok := index[key]; ok && existing.weight >= e.weight {
		return
	}
	index[key] = e
}

func (g *globSet) sort() {
	sort.SliceStable(g.others, func(i, j int) bool {
		if g.others[i].weight != g.others[j].weight {
			return g.others[i].weight > g.others[j].weight
		}
		return len(g.others[i].pattern) > len(g.others[j].pattern)
	})
}

// match follows the spec order: an exact filename beats the longest suffix,
// which beats any other pattern.
func (g *globSet) match(name string) string {
	lower := strings.ToLower(name)
	sameLength := len(lower) == len(name)

	if e, ok := g.literalsCS[name]; ok {
		return e.mime
	}
	if e, ok := g.literals[lower]; ok {
		return e.mime
	}

	start := max(0, len(name)-g.maxSuffix)
	for i := start; i < len(name); i++ {
		if e, ok := g.suffixesCS[name[i:]]; ok {
			return e.mime
		}
		suffix := strings.ToLower(name[i:])
		if sameLength {
			suffix = lower[i:]
		}
		if e, ok := g.suffixes[suffix]; ok {
			return e.mime
		}
	}

	for _, e := range g.others {
		subject := lower
		if e.cs {
			subject = name
		}
		if ok, _ := path.Match(e.pattern, subject); ok {
			return e.mime
		}
	}
	return ""
}
