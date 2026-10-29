package sshfs

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewElevation(t *testing.T) {
	tests := []struct {
		kind, user string
		want       Elevation
		ok         bool
	}{
		{"", "", Elevation{}, true},
		{"sudo", "", Elevation{Kind: "sudo"}, true},
		{"sudo", "root", Elevation{Kind: "sudo"}, true},
		{"sudo", " nix ", Elevation{Kind: "sudo", User: "nix"}, true},
		{"machinectl", "nix", Elevation{Kind: "machinectl", User: "nix"}, true},
		{"machinectl", "", Elevation{}, false},
		{"su", "", Elevation{}, false},
		{"sudo", "-oProxyCommand=x", Elevation{}, false},
		{"sudo", "a b", Elevation{}, false},
	}
	for _, tt := range tests {
		got, err := NewElevation(tt.kind, tt.user)
		assert.Equal(t, tt.ok, err == nil, "%s %q", tt.kind, tt.user)
		if tt.ok {
			assert.Equal(t, tt.want, got)
		}
	}
}

func TestElevationTagsRoundTrip(t *testing.T) {
	for _, tag := range []string{"sudo", "sudo+nix", "machinectl+nix"} {
		elevation, ok := ElevationFromTag(tag)
		require.True(t, ok, tag)
		assert.Equal(t, tag, elevation.Tag())
	}
	for _, tag := range []string{"", "root", "machinectl", "sudo+a b"} {
		_, ok := ElevationFromTag(tag)
		assert.False(t, ok, tag)
	}
}

func TestServerCommand(t *testing.T) {
	root := Elevation{Kind: "sudo"}
	nix := Elevation{Kind: "machinectl", User: "nix"}

	assert.Equal(t, "sudo -n '/usr/lib/ssh/sftp-server'", root.ServerCommand("/usr/lib/ssh/sftp-server", false))
	assert.Equal(t, "sudo -n -u nix '/usr/lib/ssh/sftp-server'", nix.ServerCommand("/usr/lib/ssh/sftp-server", false))
	assert.Equal(t, `sh -c 'if sudo -n true 2>/dev/null; then IFS= read -r _; exec sudo -n "$0"; fi; exec sudo -S -p "" "$0"' '/s'`, root.ServerCommand("/s", true))
	assert.Contains(t, nix.ServerCommand("/s", true), `exec sudo -S -p "" -u nix "$0"`)
}

func TestParseURI(t *testing.T) {
	login, dir, err := ParseURI("sftp://me@box.example.com:2222/srv/data/")
	require.NoError(t, err)
	assert.Equal(t, Login{User: "me", Host: "box.example.com", Port: 2222}, login)
	assert.Equal(t, "/srv/data", dir)
	assert.Equal(t, "sftp://me@box.example.com:2222/", login.URI())

	for _, bad := range []string{"smb://box/share", "sftp://", "sftp://-oProxy@box", "sftp://box:99999/", "sftp://-x/"} {
		_, _, err := ParseURI(bad)
		assert.Error(t, err, bad)
	}
}

// A server that is this machine: ssh runs the command locally and sudo only checks the password.
func fakeServer(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	ssh := `#!/bin/sh
while [ $# -gt 0 ]; do
  case "$1" in
    -p|-o) shift 2 ;;
    --) shift; break ;;
    -*) shift ;;
    *) break ;;
  esac
done
shift
exec sh -c "$*"
`
	sudo := `#!/bin/sh
secret=
while [ $# -gt 0 ]; do
  case "$1" in
    -n) shift ;;
    -S) secret=1; shift ;;
    -p|-u) shift 2 ;;
    *) break ;;
  esac
done
if [ -n "$secret" ]; then read -r given; [ "$given" = "hunter2" ] || exit 1; fi
if [ -z "$secret" ] && [ -z "$FAKE_SUDO_NOPASSWD" ]; then exit 1; fi
exec "$@"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ssh"), []byte(ssh), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sudo"), []byte(sudo), 0o755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
}

func canMount(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("/dev/fuse"); err != nil {
		t.Skip("no fuse")
	}
	if _, _, err := tools(); err != nil {
		t.Skip(err.Error())
	}
	if _, err := exec.LookPath("fusermount3"); err != nil {
		t.Skip("no fusermount3")
	}
	for _, candidate := range serverCandidates {
		if _, err := os.Stat(candidate); err == nil {
			return
		}
	}
	t.Skip("no sftp-server")
}

func TestMountOpensTheServerThroughSudo(t *testing.T) {
	canMount(t)
	fakeServer(t)
	files := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(files, "secret.txt"), []byte("root only"), 0o600))
	changed := make(chan struct{}, 4)
	manager := NewManager(t.TempDir(), func() { changed <- struct{}{} })
	t.Cleanup(manager.Close)
	login := Login{User: "me", Host: "box"}
	as := Elevation{Kind: "sudo"}

	_, err := manager.Mount(login, as, "")
	assert.ErrorContains(t, err, "asks for a password")
	_, err = manager.Mount(login, as, "wrong")
	assert.ErrorContains(t, err, "did not accept the password")

	mounted, err := manager.Mount(login, as, "hunter2")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(mounted.Path, files, "secret.txt"))
	require.NoError(t, err)
	assert.Equal(t, "root only", string(data))
	assert.Equal(t, "root on box (sudo)", mounted.Name)
	assert.NotEmpty(t, mounted.Home, "the home folder of root")
	assert.Equal(t, "sftp:host=box,user=me,as=sudo", filepath.Base(mounted.Path))
	assert.Equal(t, []Mounted{mounted}, manager.List())

	again, err := manager.Mount(login, as, "hunter2")
	require.NoError(t, err)
	assert.Equal(t, mounted, again, "a second connect reuses the mount")

	require.NoError(t, manager.Unmount(mounted.ID))
	assert.Empty(t, manager.List())
	_, err = os.Stat(mounted.Path)
	assert.True(t, os.IsNotExist(err), "%v", err)
}

func TestMountWithPasswordlessSudoSkipsThePassword(t *testing.T) {
	canMount(t)
	fakeServer(t)
	t.Setenv("FAKE_SUDO_NOPASSWD", "1")
	files := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(files, "a.txt"), []byte("a"), 0o600))
	manager := NewManager(t.TempDir(), nil)
	t.Cleanup(manager.Close)

	mounted, err := manager.Mount(Login{Host: "box"}, Elevation{Kind: "sudo", User: "nix"}, "ignored")

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(mounted.Path, files, "a.txt"))
	assert.Equal(t, "nix on box (sudo)", mounted.Name)
}

func TestRelaySendsThePasswordThenWhatSshfsWritesEvenOnANonBlockingPipe(t *testing.T) {
	var fds [2]int
	require.NoError(t, syscall.Pipe2(fds[:], syscall.O_NONBLOCK|syscall.O_CLOEXEC))
	reader, writer := os.NewFile(uintptr(fds[0]), "stdin"), os.NewFile(uintptr(fds[1]), "stdin-writer")
	go func() {
		time.Sleep(200 * time.Millisecond)
		_, _ = writer.WriteString("sftp bytes")
		_ = writer.Close()
	}()
	var out bytes.Buffer

	code := Relay("cat", nil, "hunter2", reader, &out, os.Stderr)

	assert.Zero(t, code)
	assert.Equal(t, "hunter2\nsftp bytes", out.String())
}
