package devices

import (
	"errors"
	"testing"

	"github.com/duxlabs-dev/dux-go/files"
	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		assert.Equal(t, code, string(files.CodeOf(err)), name)
		var coded *files.Error
		require.ErrorAs(t, err, &coded, name)
		assert.Equal(t, "boom", coded.Err.Error(), name)
	}
}

func TestWrapDBusFallsBackToTheErrorName(t *testing.T) {
	err := wrapDBus(dbus.Error{Name: BusName + ".Error.Failed"})
	assert.ErrorContains(t, err, BusName+".Error.Failed")
}

func TestCodeOfPlainErrorsIsIO(t *testing.T) {
	assert.Equal(t, files.CodeIO, files.CodeOf(wrapDBus(errors.New("broken"))))
	assert.Nil(t, wrapDBus(nil))
}
