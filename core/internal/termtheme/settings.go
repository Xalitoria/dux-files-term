package termtheme

import (
	"strconv"
	"strings"
)

// What a terminal does when a program rings the bell.
const (
	BellAudio     = "audio"
	BellVisual    = "visual"
	BellAttention = "attention"
)

// UnlimitedScrollback stands for a terminal that keeps every line.
const UnlimitedScrollback = -1

var quietBell = []string{}

func bellOf(audio, visual, attention bool) []string {
	bell := []string{}
	if audio {
		bell = append(bell, BellAudio)
	}
	if visual {
		bell = append(bell, BellVisual)
	}
	if attention {
		bell = append(bell, BellAttention)
	}
	return bell
}

func (p *Profile) setSettingDefaults(scrollback int, wheel float64, bell []string) {
	if p.ScrollbackLines == 0 {
		p.ScrollbackLines = scrollback
	}
	if p.WheelLines == 0 {
		p.WheelLines = wheel
	}
	if p.Bell == nil {
		p.Bell = bell
	}
}

func anySet(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func boolOr(value string, fallback bool) bool {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return truthy(value)
}

// A line count. Negative numbers, "unlimited" and "infinite" keep every line.
func parseLines(text string) int {
	text = strings.ToLower(strings.Trim(strings.TrimSpace(text), `"'`))
	switch {
	case text == "":
		return 0
	case text == "unlimited", text == "infinite", strings.HasPrefix(text, "-"):
		return UnlimitedScrollback
	}
	lines, err := strconv.Atoi(number.FindString(text))
	if err != nil || lines == 0 {
		return 0
	}
	return lines
}

// A positive number such as a wheel multiplier or a duration; anything else is 0.
func parsePositive(text string) float64 {
	value, err := strconv.ParseFloat(strings.Trim(strings.TrimSpace(text), `"'`), 64)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}
