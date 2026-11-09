package terminal

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func currentMatch(t *testing.T, f frame) match {
	t.Helper()
	var current []match
	for _, m := range f.Matches {
		if m.Current {
			current = append(current, m)
		}
	}
	require.Len(t, current, 1, "one current match on screen")
	return current[0]
}

func TestSearchWalksTheHistory(t *testing.T) {
	script := `i=1; while [ $i -le 60 ]; do echo "Line $i"; i=$((i+1)); done; echo done; sleep 5`
	session, err := startSession("terminal:search", []string{"sh", "-c", script}, t.TempDir(), 80, 24, false, nil, (&published{}).publish)
	require.NoError(t, err)
	t.Cleanup(session.Close)
	require.Eventually(t, func() bool { return strings.Contains(screenText(session), "done") }, 5*time.Second, 20*time.Millisecond)
	// 62 lines were printed, so Line 1 to Line 38 sit in the history and Line 39 is the top screen row.

	count, index := session.Search("LINE 5", 0)
	assert.Equal(t, 11, count, "Line 5 and Line 50 to Line 59")
	assert.Equal(t, 11, index, "a fresh search picks the match nearest the bottom of the view")
	f := frameOf(session)
	assert.Equal(t, 0, f.Scroll)
	assert.Equal(t, match{Y: 20, X: 0, W: 6, Current: true}, currentMatch(t, f))
	assert.Len(t, f.Matches, 10, "Line 50 to Line 59 are on screen")

	_, index = session.Search("line 5", -1)
	assert.Equal(t, 10, index)
	assert.Equal(t, 0, frameOf(session).Scroll, "Line 58 is on screen already")

	for range 9 {
		_, index = session.Search("line 5", -1)
	}
	assert.Equal(t, 1, index)
	f = frameOf(session)
	assert.Equal(t, 34, f.Scroll, "Line 5 scrolls to the top of the view")
	assert.Equal(t, match{Y: 0, X: 0, W: 6, Current: true}, currentMatch(t, f))

	_, index = session.Search("line 5", -1)
	assert.Equal(t, 11, index, "moving above the first match wraps to the last")
	f = frameOf(session)
	assert.Equal(t, 3, f.Scroll, "Line 59 scrolls to the bottom of the view")
	assert.Equal(t, 23, currentMatch(t, f).Y)

	_, index = session.Search("line 5", 1)
	assert.Equal(t, 1, index, "moving below the last match wraps to the first")

	count, index = session.Search("nothing like this", 0)
	assert.Equal(t, 0, count)
	assert.Equal(t, 0, index)
	assert.Empty(t, frameOf(session).Matches)

	count, index = session.Search("", 0)
	assert.Equal(t, 0, count)
	assert.Equal(t, 0, index)
	assert.Nil(t, frameOf(session).Matches)
}

func TestSearchMeasuresWideCells(t *testing.T) {
	session, err := startSession("terminal:wide", []string{"sh", "-c", `printf 'ab 日本語 cd'; sleep 5`}, t.TempDir(), 40, 5, false, nil, (&published{}).publish)
	require.NoError(t, err)
	t.Cleanup(session.Close)
	require.Eventually(t, func() bool { return strings.Contains(screenText(session), "cd") }, 5*time.Second, 20*time.Millisecond)

	count, _ := session.Search("本語 c", 0)
	assert.Equal(t, 1, count)
	assert.Equal(t, match{Y: 0, X: 5, W: 6, Current: true}, currentMatch(t, frameOf(session)))
}
