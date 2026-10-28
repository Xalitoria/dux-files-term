package termtheme

import "strings"

// cursorShape maps the names terminals use to "block", "bar" or "underline".
func cursorShape(name string) string {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(name), `"'`)) {
	case "block", "steadyblock", "blinkingblock", "block_hollow":
		return "block"
	case "bar", "beam", "ibeam", "steadybar", "blinkingbar":
		return "bar"
	case "underline", "steadyunderline", "blinkingunderline":
		return "underline"
	}
	return ""
}

func truthy(value string) bool {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(value), `"'`)) {
	case "true", "yes", "on", "1", "always":
		return true
	}
	return false
}

func (c *Profile) setCursor(shape string, blink bool) {
	c.CursorShape, c.CursorBlink = cursorShape(shape), blink
}
