package terminal

import (
	"strings"
	"testing"
	"time"

	"github.com/duxlabs-dev/dux-files/core/internal/termtheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBellReachesOneFrame(t *testing.T) {
	events := &published{}
	session, err := startSession("terminal:bell", []string{"sh", "-c", `printf 'ding\a'; sleep 0.3; printf ' dong'; sleep 5`}, t.TempDir(), 40, 5, false, nil, events.publish)
	require.NoError(t, err)
	t.Cleanup(session.Close)

	require.Eventually(t, func() bool { return strings.Contains(screenText(session), "dong") }, 5*time.Second, 20*time.Millisecond)
	require.Eventually(t, func() bool { return events.bellCount() > 0 }, time.Second, 20*time.Millisecond)
	assert.Equal(t, 1, events.bellCount(), "the bell is reported in one frame only")
}

func TestScrollbackFollowsTheProfile(t *testing.T) {
	assert.Equal(t, scrollbackLines, scrollbackSize(0), "unset keeps the dock's size")
	assert.Equal(t, 10, scrollbackSize(10))
	assert.Equal(t, maxScrollbackLines, scrollbackSize(termtheme.UnlimitedScrollback))
	assert.Equal(t, maxScrollbackLines, scrollbackSize(maxScrollbackLines*3), "huge sizes are capped")

	script := `i=1; while [ $i -le 60 ]; do echo "Line $i"; i=$((i+1)); done; echo done; sleep 5`
	profile := termtheme.Profile{Opacity: 1, ScrollbackLines: 10}
	session, err := startSession("terminal:history", []string{"sh", "-c", script}, t.TempDir(), 80, 24, false, &profile, (&published{}).publish)
	require.NoError(t, err)
	t.Cleanup(session.Close)
	require.Eventually(t, func() bool { return strings.Contains(screenText(session), "done") }, 5*time.Second, 20*time.Millisecond)

	assert.Equal(t, 10, frameOf(session).Scrollback)
}
