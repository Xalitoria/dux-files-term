package devices

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

// wrapDBus maps the udisks error names onto the codes the UI already knows.
func wrapDBus(err error) error {
	if err == nil {
		return nil
	}
	call, ok := err.(dbus.Error)
	if !ok {
		return fail("EIO", "%s", err)
	}
	return &Error{Code: codeForName(call.Name), Message: message(call)}
}

func codeForName(name string) string {
	switch strings.TrimPrefix(name, BusName+".Error.") {
	case "NotAuthorized", "NotAuthorizedCanObtain", "NotAuthorizedDismissed":
		return "EACCES"
	case "DeviceBusy":
		return "EBUSY"
	case "AlreadyMounted", "AlreadyUnmounting":
		return "EEXIST"
	case "NotMounted", "OptionNotPermitted", "MountedByOtherUser":
		return "EINVAL"
	case "NotSupported":
		return "NOTSUPPORTED"
	}
	switch name {
	case "org.freedesktop.DBus.Error.ServiceUnknown", "org.freedesktop.DBus.Error.NameHasNoOwner":
		return "NOTSUPPORTED"
	case "org.freedesktop.DBus.Error.AccessDenied":
		return "EACCES"
	case "org.freedesktop.DBus.Error.UnknownMethod", "org.freedesktop.DBus.Error.UnknownInterface":
		return "NOTSUPPORTED"
	case "org.freedesktop.DBus.Error.UnknownObject":
		return "ENOENT"
	}
	return "EIO"
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
