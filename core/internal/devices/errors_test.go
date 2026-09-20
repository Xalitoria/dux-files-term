package devices

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/assert"
)

func TestWrapDBusMapsUdisksErrorsToCodes(t *testing.T) {
	cases := map[string]string{
		BusName + ".Error.NotAuthorizedDismissed":   "EACCES",
		BusName + ".Error.DeviceBusy":               "EBUSY",
		BusName + ".Error.AlreadyMounted":           "EEXIST",
		BusName + ".Error.NotMounted":               "EINVAL",
		BusName + ".Error.NotSupported":             "NOTSUPPORTED",
		BusName + ".Error.Failed":                   "EIO",
		"org.freedesktop.DBus.Error.ServiceUnknown": "NOTSUPPORTED",
		"org.freedesktop.DBus.Error.AccessDenied":   "EACCES",
		"org.freedesktop.DBus.Error.UnknownObject":  "ENOENT",
		"org.freedesktop.DBus.Error.UnknownMethod":  "NOTSUPPORTED",
	}

	for name, code := range cases {
		err := wrapDBus(dbus.Error{Name: name, Body: []any{"boom"}})
		assert.Equal(t, code, CodeOf(err), name)
		assert.Equal(t, "boom", err.Error(), name)
	}
}

func TestWrapDBusFallsBackToTheErrorName(t *testing.T) {
	err := wrapDBus(dbus.Error{Name: BusName + ".Error.Failed"})
	assert.Equal(t, BusName+".Error.Failed", err.Error())
}

func TestCodeOfPlainErrorsIsIO(t *testing.T) {
	assert.Equal(t, "EIO", CodeOf(errors.New("broken")))
	assert.Nil(t, wrapDBus(nil))
}
