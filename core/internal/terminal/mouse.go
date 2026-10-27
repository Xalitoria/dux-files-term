package terminal

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

var mouseButtons = map[string]uv.MouseButton{
	"none":      uv.MouseNone,
	"left":      uv.MouseLeft,
	"middle":    uv.MouseMiddle,
	"right":     uv.MouseRight,
	"wheelUp":   uv.MouseWheelUp,
	"wheelDown": uv.MouseWheelDown,
}

// Levels the UI sees: 1 reports clicks, 2 adds drags, 3 adds plain motion.
var mouseLevels = map[ansi.Mode]int{
	ansi.ModeMouseX10:         1,
	ansi.ModeMouseNormal:      1,
	ansi.ModeMouseHighlight:   1,
	ansi.ModeMouseButtonEvent: 2,
	ansi.ModeMouseAnyEvent:    3,
}

type mouseTracker struct {
	on map[ansi.Mode]bool
}

func (t *mouseTracker) set(mode ansi.Mode, enabled bool) {
	if _, tracked := mouseLevels[mode]; !tracked {
		return
	}
	if t.on == nil {
		t.on = map[ansi.Mode]bool{}
	}
	t.on[mode] = enabled
}

func (t *mouseTracker) level() int {
	level := 0
	for mode, enabled := range t.on {
		if enabled {
			level = max(level, mouseLevels[mode])
		}
	}
	return level
}

func keyMods(mods int) uv.KeyMod {
	var out uv.KeyMod
	if mods&modShift != 0 {
		out |= uv.ModShift
	}
	if mods&modAlt != 0 {
		out |= uv.ModAlt
	}
	if mods&modCtrl != 0 {
		out |= uv.ModCtrl
	}
	return out
}

func mouseEventOf(in input) uv.MouseEvent {
	mouse := uv.Mouse{X: in.x, Y: in.y, Button: mouseButtons[in.button], Mod: keyMods(in.mods)}
	switch in.mouse {
	case "press":
		return uv.MouseClickEvent(mouse)
	case "release":
		return uv.MouseReleaseEvent(mouse)
	case "move":
		return uv.MouseMotionEvent(mouse)
	case "wheel":
		return uv.MouseWheelEvent(mouse)
	}
	return nil
}
