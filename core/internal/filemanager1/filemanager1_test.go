package filemanager1

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const busConfig = `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:dir=%s</listen>
  <auth>EXTERNAL</auth>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>`

func privateBus(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon not installed")
	}
	dir := t.TempDir()
	config := filepath.Join(dir, "bus.conf")
	require.NoError(t, os.WriteFile(config, []byte(strings.Replace(busConfig, "%s", dir, 1)), 0o600))

	cmd := exec.Command(bin, "--config-file="+config, "--nofork", "--nopidfile", "--print-address")
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	address, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	return strings.TrimSpace(address)
}

func connect(t *testing.T, address string) *dbus.Conn {
	t.Helper()
	conn, err := dbus.Connect(address)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

type recorder struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (r *recorder) record(method string, uris []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, method+" "+strings.Join(uris, " "))
	return r.err
}

func (r *recorder) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func (r *recorder) ShowFolders(uris []string) error { return r.record("folders", uris) }

func (r *recorder) ShowItems(uris []string) error { return r.record("items", uris) }

func (r *recorder) ShowItemProperties(uris []string) error { return r.record("properties", uris) }

func start(t *testing.T, address string, handler Handler) *Service {
	t.Helper()
	conn, err := dbus.Connect(address)
	require.NoError(t, err)
	svc, err := New(conn, handler)
	require.NoError(t, err)
	t.Cleanup(svc.Close)
	return svc
}

func TestIntrospectionMatchesTheInterface(t *testing.T) {
	address := privateBus(t)
	svc := start(t, address, &recorder{})
	require.True(t, svc.SetEnabled(true).Owned)

	node, err := introspect.Call(connect(t, address).Object(Name, Path))
	require.NoError(t, err)

	signatures := map[string]string{}
	for _, iface := range node.Interfaces {
		if iface.Name != Interface {
			continue
		}
		for _, method := range iface.Methods {
			var args []string
			for _, arg := range method.Args {
				args = append(args, arg.Direction+":"+arg.Type)
			}
			signatures[method.Name] = strings.Join(args, ",")
		}
	}
	assert.Equal(t, map[string]string{
		"ShowFolders":        "in:as,in:s",
		"ShowItems":          "in:as,in:s",
		"ShowItemProperties": "in:as,in:s",
	}, signatures)
}

func TestMethodsReachTheHandler(t *testing.T) {
	address := privateBus(t)
	rec := &recorder{}
	svc := start(t, address, rec)
	require.True(t, svc.SetEnabled(true).Owned)

	obj := connect(t, address).Object(Name, Path)
	uris := []string{"file:///home/me/a.txt", "file:///home/me/b%20c.txt"}
	require.NoError(t, obj.Call(Interface+".ShowFolders", 0, uris[:1], "").Err)
	require.NoError(t, obj.Call(Interface+".ShowItems", 0, uris, "startup-token").Err)
	require.NoError(t, obj.Call(Interface+".ShowItemProperties", 0, uris[1:], "").Err)

	assert.Equal(t, []string{
		"folders file:///home/me/a.txt",
		"items file:///home/me/a.txt file:///home/me/b%20c.txt",
		"properties file:///home/me/b%20c.txt",
	}, rec.seen())
}

func TestHandlerErrorsReachTheCaller(t *testing.T) {
	address := privateBus(t)
	svc := start(t, address, &recorder{err: errors.New("/nope does not exist")})
	require.True(t, svc.SetEnabled(true).Owned)

	err := connect(t, address).Object(Name, Path).Call(Interface+".ShowItems", 0, []string{"file:///nope"}, "").Err
	var dbusErr dbus.Error
	require.ErrorAs(t, err, &dbusErr)
	assert.Equal(t, "org.freedesktop.DBus.Error.Failed", dbusErr.Name)
	assert.Contains(t, dbusErr.Error(), "/nope does not exist")
}

func TestAnotherOwnerIsNotReplaced(t *testing.T) {
	address := privateBus(t)
	other := connect(t, address)
	reply, err := other.RequestName(Name, 0)
	require.NoError(t, err)
	require.Equal(t, dbus.RequestNameReplyPrimaryOwner, reply)

	svc := start(t, address, &recorder{})
	status := svc.SetEnabled(true)
	assert.True(t, status.Enabled)
	assert.False(t, status.Owned)
	assert.True(t, status.Queued)
	assert.NotEmpty(t, status.Owner)

	_, err = other.ReleaseName(Name)
	require.NoError(t, err)
	assert.True(t, svc.Status().Owned, "the queued request takes over once the holder lets go")
}

func TestDisablingReleasesTheName(t *testing.T) {
	address := privateBus(t)
	svc := start(t, address, &recorder{})
	require.True(t, svc.SetEnabled(true).Owned)

	status := svc.SetEnabled(false)
	assert.False(t, status.Enabled)
	assert.False(t, status.Owned)

	reply, err := connect(t, address).RequestName(Name, dbus.NameFlagDoNotQueue)
	require.NoError(t, err)
	assert.Equal(t, dbus.RequestNameReplyPrimaryOwner, reply)
}

func TestDisabledServiceNeverRequestsTheName(t *testing.T) {
	address := privateBus(t)
	svc := start(t, address, &recorder{})

	status := svc.Status()
	assert.False(t, status.Enabled)
	assert.False(t, status.Owned)
	assert.False(t, status.Queued)
	assert.Empty(t, status.Owner)
}
