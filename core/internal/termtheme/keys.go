package termtheme

import (
	"slices"
	"strings"
)

// The actions a docked terminal can bind.
const (
	ActionCopy      = "copy"
	ActionPaste     = "paste"
	ActionZoomIn    = "zoomIn"
	ActionZoomOut   = "zoomOut"
	ActionZoomReset = "zoomReset"
)

var ctrlZoomKeys = map[string][]string{
	ActionZoomIn:    {"ctrl+plus"},
	ActionZoomOut:   {"ctrl+minus"},
	ActionZoomReset: {"ctrl+0"},
}

var modifiers = map[string]string{
	"ctrl": "ctrl", "control": "ctrl", "ctl": "ctrl",
	"alt": "alt", "opt": "alt", "option": "alt", "mod1": "alt",
	"shift": "shift",
	"super": "super", "cmd": "super", "command": "super", "win": "super", "mod4": "super", "meta": "super",
}

var modifierOrder = []string{"ctrl", "alt", "shift", "super"}

var keyAliases = map[string]string{
	"=": "plus", "+": "plus", "-": "minus",
	"equal": "plus", "equals": "plus", "plus": "plus", "add": "plus", "kpadd": "plus", "numpadadd": "plus",
	"minus": "minus", "underscore": "minus", "subtract": "minus", "kpsubtract": "minus", "numpadsubtract": "minus",
	"return": "enter", "esc": "escape", "back": "backspace",
	"prior": "pageup", "pgup": "pageup", "next": "pagedown", "pgdn": "pagedown",
}

// A chord reads as "ctrl+shift+c": modifiers in a fixed order, then the key.
// The same text comes out of every terminal's spelling, and the dock builds its own from key events.
func chordOf(mods []string, key string) (string, bool) {
	key = normalizeKey(key)
	if key == "" || strings.HasPrefix(key, "xf86") {
		return "", false
	}
	var held []string
	for _, mod := range mods {
		if strings.TrimSpace(mod) == "" {
			continue
		}
		name, ok := modifiers[strings.ToLower(strings.TrimSpace(mod))]
		if !ok {
			return "", false
		}
		held = append(held, name)
	}
	var parts []string
	for _, mod := range modifierOrder {
		if slices.Contains(held, mod) {
			parts = append(parts, mod)
		}
	}
	return strings.Join(append(parts, key), "+"), true
}

// Parses "ctrl+shift+c". A trailing "+" is the plus key. A sequence such as "ctrl+a>c" is not a chord.
func parseChord(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" || strings.Contains(text, ">") {
		return "", false
	}
	key := ""
	switch {
	case text == "+":
		key, text = "plus", ""
	case strings.HasSuffix(text, "++"):
		key, text = "plus", strings.TrimSuffix(text, "++")
	}
	parts := strings.Split(text, "+")
	if key == "" {
		key, parts = parts[len(parts)-1], parts[:len(parts)-1]
	}
	return chordOf(parts, key)
}

func normalizeKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if alias, ok := keyAliases[name]; ok {
		return alias
	}
	name = strings.NewReplacer("_", "", "-", "").Replace(name)
	name = strings.TrimPrefix(name, "arrow")
	if alias, ok := keyAliases[name]; ok {
		return alias
	}
	for _, prefix := range []string{"key", "kp", "numpad"} {
		if rest := strings.TrimPrefix(name, prefix); len(rest) == 1 && rest != name {
			return rest
		}
	}
	return name
}

func (p *Profile) bind(action, chord string) {
	if chord == "" || slices.Contains(p.Keys[action], chord) {
		return
	}
	if p.Keys == nil {
		p.Keys = map[string][]string{}
	}
	p.Keys[action] = append(p.Keys[action], chord)
}

func (p *Profile) addKeys(keys map[string][]string) {
	for action, chords := range keys {
		for _, chord := range chords {
			p.bind(action, chord)
		}
	}
}
