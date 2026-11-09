package terminal

import (
	"math"
	"sort"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

type match struct {
	Y       int  `json:"y"`
	X       int  `json:"x"`
	W       int  `json:"w"`
	Current bool `json:"current,omitempty"`
}

// A place in the history and screen together; line 0 is the oldest history line.
type spot struct {
	line, x, w int
}

func (a spot) before(b spot) bool {
	return a.line < b.line || (a.line == b.line && a.x < b.x)
}

type searchState struct {
	query   string
	current spot
	found   bool
}

// Search finds the text, ignoring case, in the history and screen and scrolls the current match into view.
// Step -1 moves to the match above the current one and 1 to the one below; 0 starts over from the bottom of the view.
// It returns how many matches there are and which one is current, counting from 1.
func (s *Session) Search(query string, step int) (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, 0
	}
	defer s.scheduleLocked()
	query = strings.ToLower(query)
	if query == "" {
		s.search = searchState{}
		return 0, 0
	}
	history, height := s.historyLocked(), s.emu.Height()
	all := s.matchesLocked(query, 0, history+height, history)
	if len(all) == 0 {
		s.search = searchState{query: query}
		return 0, 0
	}
	index := s.pickLocked(all, query, step, history, height)
	s.search = searchState{query: query, current: all[index], found: true}
	s.revealLocked(all[index].line, history, height)
	return len(all), index + 1
}

func (s *Session) pickLocked(all []spot, query string, step, history, height int) int {
	fresh := s.search.query != query || !s.search.found || step == 0
	switch {
	case fresh:
		bottom := spot{line: history - s.scroll + height - 1, x: math.MaxInt}
		if i := lastBefore(all, bottom); i >= 0 {
			return i
		}
		return len(all) - 1
	case step < 0:
		if i := lastBefore(all, s.search.current); i >= 0 {
			return i
		}
		return len(all) - 1
	default:
		i := sort.Search(len(all), func(i int) bool { return s.search.current.before(all[i]) })
		if i == len(all) {
			return 0
		}
		return i
	}
}

// The index of the last spot before at, or -1.
func lastBefore(all []spot, at spot) int {
	return sort.Search(len(all), func(i int) bool { return !all[i].before(at) }) - 1
}

func (s *Session) revealLocked(line, history, height int) {
	top := history - s.scroll
	switch {
	case line < top:
		s.scroll = history - line
	case line >= top+height:
		s.scroll = max(0, history-line+height-1)
	}
}

func (s *Session) historyLocked() int {
	if s.emu.IsAltScreen() {
		return 0
	}
	return s.emu.ScrollbackLen()
}

func (s *Session) visibleMatchesLocked(history, height int) []match {
	if s.search.query == "" {
		return nil
	}
	top := history - s.scroll
	var matches []match
	for _, found := range s.matchesLocked(s.search.query, top, top+height, history) {
		matches = append(matches, match{Y: found.line - top, X: found.x, W: found.w, Current: s.search.found && found == s.search.current})
	}
	return matches
}

// The matches on lines from up to to, with the query already lowered.
func (s *Session) matchesLocked(query string, from, to, history int) []spot {
	width := s.emu.Width()
	var found []spot
	var cols []int
	for line := from; line < to; line++ {
		var text string
		text, cols = lineText(width, s.lineSource(line, history), cols[:0])
		for start := 0; ; {
			i := strings.Index(text[start:], query)
			if i < 0 {
				break
			}
			i += start
			end := i + len(query)
			found = append(found, spot{line: line, x: cols[i], w: max(1, cols[end]-cols[i])})
			start = end
		}
	}
	return found
}

// The lowered text of a line and the column each byte sits in, with one more entry closing the last cell.
func lineText(width int, cellAt func(x int) *uv.Cell, cols []int) (string, []int) {
	var text strings.Builder
	x := 0
	for x < width {
		cell := cellAt(x)
		if cell == nil || cell.Content == "" {
			cell = &uv.EmptyCell
		}
		content := strings.ToLower(cell.Content)
		text.WriteString(content)
		for range len(content) {
			cols = append(cols, x)
		}
		x += max(cell.Width, 1)
	}
	return text.String(), append(cols, x)
}
