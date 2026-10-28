package termtheme

import (
	"image/color"
	"strconv"
	"strings"
)

// Accepts #rgb, #rrggbb, #rrrrggggbbbb, 0xrrggbb, rrggbb and rgb:rr/gg/bb.
func parseColor(text string) (color.Color, bool) {
	text = strings.Trim(strings.TrimSpace(text), `"'`)
	switch {
	case strings.HasPrefix(text, "rgb:"):
		text = strings.ReplaceAll(strings.TrimPrefix(text, "rgb:"), "/", "")
	case strings.HasPrefix(text, "0x"), strings.HasPrefix(text, "0X"):
		text = text[2:]
	default:
		text = strings.TrimPrefix(text, "#")
	}
	switch len(text) {
	case 3:
		text = string([]byte{text[0], text[0], text[1], text[1], text[2], text[2]})
	case 12:
		text = text[0:2] + text[4:6] + text[8:10]
	}
	if len(text) != 6 {
		return nil, false
	}
	value, err := strconv.ParseUint(text, 16, 32)
	if err != nil {
		return nil, false
	}
	return color.RGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 0xff}, true
}

func tint(text string) color.Color {
	c, _ := parseColor(text)
	return c
}

func parseRGBTriple(text string) (color.Color, bool) {
	parts := strings.Split(text, ",")
	if len(parts) != 3 {
		return nil, false
	}
	var channels [3]uint8
	for i, part := range parts {
		value, err := strconv.ParseUint(strings.TrimSpace(part), 10, 8)
		if err != nil {
			return nil, false
		}
		channels[i] = uint8(value)
	}
	return color.RGBA{R: channels[0], G: channels[1], B: channels[2], A: 0xff}, true
}

func parseOpacity(text string) (float64, bool) {
	value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return 0, false
	}
	return clampOpacity(value), true
}

type ini map[string]map[string]string

// Sections keep their name without brackets, so [[default]] reads as "default".
func parseINI(data []byte) ini {
	out := ini{}
	section := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "", strings.HasPrefix(line, "#"), strings.HasPrefix(line, ";"):
			continue
		case strings.HasPrefix(line, "["):
			section = strings.Trim(line, "[] ")
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		if out[section] == nil {
			out[section] = map[string]string{}
		}
		out[section][strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return out
}

func (i ini) get(section, key string) string {
	return i[section][key]
}

func firstField(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// Generic family names pick no font, so there is nothing to follow.
func namedFamily(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "auto", "monospace", "mono", "default", "system-ui":
		return ""
	}
	return strings.TrimSpace(name)
}
