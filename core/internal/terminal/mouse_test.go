package terminal

import (
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestMouseTrackerLevel(t *testing.T) {
	tests := []struct {
		name  string
		steps []struct {
			mode    ansi.Mode
			enabled bool
		}
		want int
	}{
		{"nothing enabled", nil, 0},
		{"clicks", []struct {
			mode    ansi.Mode
			enabled bool
		}{{ansi.ModeMouseNormal, true}}, 1},
		{"highest mode wins", []struct {
			mode    ansi.Mode
			enabled bool
		}{{ansi.ModeMouseNormal, true}, {ansi.ModeMouseAnyEvent, true}}, 3},
		{"falls back when the highest is disabled", []struct {
			mode    ansi.Mode
			enabled bool
		}{{ansi.ModeMouseNormal, true}, {ansi.ModeMouseButtonEvent, true}, {ansi.ModeMouseButtonEvent, false}}, 1},
		{"unrelated modes are ignored", []struct {
			mode    ansi.Mode
			enabled bool
		}{{ansi.ModeMouseExtSgr, true}, {ansi.ModeAltScreen, true}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tracker mouseTracker
			for _, step := range tt.steps {
				tracker.set(step.mode, step.enabled)
			}
			assert.Equal(t, tt.want, tracker.level())
		})
	}
}

func TestMouseEventOf(t *testing.T) {
	tests := []struct {
		name string
		in   input
		want uv.MouseEvent
	}{
		{"press", input{mouse: "press", x: 3, y: 4, button: "left"}, uv.MouseClickEvent{X: 3, Y: 4, Button: uv.MouseLeft}},
		{"release with modifiers", input{mouse: "release", x: 1, y: 2, button: "right", mods: modShift | modCtrl}, uv.MouseReleaseEvent{X: 1, Y: 2, Button: uv.MouseRight, Mod: uv.ModShift | uv.ModCtrl}},
		{"hover has no button", input{mouse: "move", x: 5, y: 6, button: "none"}, uv.MouseMotionEvent{X: 5, Y: 6}},
		{"wheel", input{mouse: "wheel", button: "wheelDown", mods: modAlt}, uv.MouseWheelEvent{Button: uv.MouseWheelDown, Mod: uv.ModAlt}},
		{"unknown kind", input{mouse: "tap"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mouseEventOf(tt.in))
		})
	}
}

func TestSessionReportsMouseToProgram(t *testing.T) {
	session, _ := startShell(t, &published{})
	level := func() int {
		session.mu.Lock()
		defer session.mu.Unlock()
		return session.mouse.level()
	}

	session.Input(0, input{text: `printf '\033[?1002h\033[?1006h'; cat >/dev/null`})
	session.Input(1, input{key: "enter"})
	assert.Eventually(t, func() bool { return level() == 2 }, 5*time.Second, 20*time.Millisecond)

	session.Input(2, input{mouse: "press", x: 5, y: 2, button: "left"})
	assert.Eventually(t, func() bool { return strings.Contains(screenText(session), "^[[<0;6;3M") }, 5*time.Second, 20*time.Millisecond)
}
