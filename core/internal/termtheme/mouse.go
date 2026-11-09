package termtheme

import "strings"

// What a right or middle click does in the terminal view.
const (
	ClickMenu         = "menu"
	ClickExtend       = "extend"
	ClickPaste        = "paste"
	ClickPrimaryPaste = "primaryPaste"
	ClickCopy         = "copy"
	ClickCopyOrPaste  = "copyOrPaste"
	ClickNone         = "none"
)

func (p *Profile) setClickDefaults(right, middle string) {
	if p.RightClick == "" {
		p.RightClick = right
	}
	if p.MiddleClick == "" {
		p.MiddleClick = middle
	}
}

func ghosttyClick(action string) string {
	switch strings.TrimSpace(action) {
	case "context-menu":
		return ClickMenu
	case "paste", "clipboard-paste":
		return ClickPaste
	case "primary-paste":
		return ClickPrimaryPaste
	case "copy":
		return ClickCopy
	case "copy-or-paste":
		return ClickCopyOrPaste
	case "ignore":
		return ClickNone
	}
	return ""
}

func kittyClick(action []string) string {
	if len(action) == 0 {
		return ClickNone
	}
	switch action[0] {
	case "mouse_selection":
		if len(action) > 1 && (action[1] == "extend" || action[1] == "move-end") {
			return ClickExtend
		}
	case "paste_from_clipboard":
		return ClickPaste
	case "paste_from_selection":
		return ClickPrimaryPaste
	case "copy_to_clipboard":
		return ClickCopy
	}
	return ClickNone
}

func alacrittyClick(action string) string {
	switch action {
	case "Paste":
		return ClickPaste
	case "PasteSelection":
		return ClickPrimaryPaste
	case "ExpandSelection":
		return ClickExtend
	case "Copy":
		return ClickCopy
	}
	return ClickNone
}

var footDefaultButtons = map[string]string{"BTN_RIGHT": "select-extend", "BTN_MIDDLE": "primary-paste"}

func footClick(action string) string {
	switch action {
	case "select-extend", "select-extend-character-wise":
		return ClickExtend
	case "clipboard-paste":
		return ClickPaste
	case "primary-paste":
		return ClickPrimaryPaste
	}
	return ClickNone
}

// A default button stays unset until the config rebinds it, so the terminal's own default applies.
func footClicks(section map[string]string) (right, middle string) {
	bound := map[string]string{}
	for action, value := range section {
		for _, combo := range strings.Fields(value) {
			bound[combo] = action
		}
	}
	click := func(button string) string {
		switch action, ok := bound[button]; {
		case ok:
			return footClick(action)
		case section[footDefaultButtons[button]] != "":
			return ClickNone
		}
		return ""
	}
	return click("BTN_RIGHT"), click("BTN_MIDDLE")
}
