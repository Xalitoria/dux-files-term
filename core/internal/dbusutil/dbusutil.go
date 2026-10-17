package dbusutil

import (
	"bytes"

	"github.com/godbus/dbus/v5"
)

// udisks and gvfs terminate every ay string with a NUL.
func ByteString(raw []byte) string {
	return string(bytes.TrimRight(raw, "\x00"))
}

func ErrorMessage(call dbus.Error) string {
	if len(call.Body) == 0 {
		return call.Name
	}
	text, ok := call.Body[0].(string)
	if !ok || text == "" {
		return call.Name
	}
	return text
}
