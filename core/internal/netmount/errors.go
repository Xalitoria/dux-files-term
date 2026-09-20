package netmount

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func fail(code, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func CodeOf(err error) string {
	if coded, ok := err.(*Error); ok {
		return coded.Code
	}
	return "EIO"
}

// gvfs answers with unmapped GError names, so the message text decides what
// the error name cannot.
func wrapDBus(err error) error {
	if err == nil {
		return nil
	}
	call, ok := err.(dbus.Error)
	if !ok {
		return fail("EIO", "%s", err)
	}

	text := message(call)
	switch {
	case strings.Contains(call.Name, "PermissionDenied"), strings.Contains(text, "Permission denied"):
		return &Error{Code: "EACCES", Message: text}
	case strings.Contains(text, "dialog cancelled"), strings.Contains(text, "cancelled"):
		return &Error{Code: "ECANCELED", Message: text}
	case strings.Contains(text, "already mounted"):
		return &Error{Code: "EEXIST", Message: text}
	case strings.Contains(text, "not mounted"):
		return &Error{Code: "EINVAL", Message: text}
	case strings.Contains(text, "busy"):
		return &Error{Code: "EBUSY", Message: text}
	case call.Name == "org.freedesktop.DBus.Error.ServiceUnknown":
		return &Error{Code: "NOTSUPPORTED", Message: "gvfs is not running"}
	}
	return &Error{Code: "EIO", Message: text}
}

func message(call dbus.Error) string {
	if len(call.Body) == 0 {
		return call.Name
	}
	text, ok := call.Body[0].(string)
	if !ok || text == "" {
		return call.Name
	}
	return text
}
