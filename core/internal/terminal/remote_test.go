package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/duxlabs-dev/dux-files/core/internal/sshfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRemote(t *testing.T) {
	tests := []struct {
		dir  string
		want remote
		ok   bool
	}{
		{"/run/user/1000/gvfs/sftp:host=box.example.com,user=me/srv/my%20dir", remote{host: "box.example.com", user: "me", dir: "/srv/my%20dir"}, true},
		{"/run/user/1000/gvfs/sftp:host=box,port=2222", remote{host: "box", port: 2222, dir: "/"}, true},
		{"/run/user/1000/gvfs/ssh:host=%5B::1%5D,user=a.b-c/", remote{host: "::1", user: "a.b-c", dir: "/"}, true},
		{"/run/user/1000/gvfs/sftp:host=-oProxyCommand=x", remote{}, false},
		{"/run/user/1000/gvfs/sftp:host=box,user=-x", remote{}, false},
		{"/run/user/1000/gvfs/sftp:host=box,port=99999", remote{}, false},
		{"/run/user/1000/gvfs/smb-share:server=x,share=y", remote{}, false},
		{"/home/me/sftp:host=box", remote{host: "box", dir: "/"}, true},
		{"/run/user/1000/dfiles/sshfs/sftp:host=box,user=me,as=sudo/etc", remote{host: "box", user: "me", dir: "/etc", as: sshfs.Elevation{Kind: "sudo"}}, true},
		{"/run/user/1000/dfiles/sshfs/sftp:host=box,as=machinectl+nix/", remote{host: "box", dir: "/", as: sshfs.Elevation{Kind: "machinectl", User: "nix"}}, true},
		{"/run/user/1000/dfiles/sshfs/sftp:host=box,as=sudo+-x/", remote{}, false},
		{"/home/me", remote{}, false},
	}
	for _, tt := range tests {
		got, ok := parseRemote(tt.dir)
		assert.Equal(t, tt.ok, ok, tt.dir)
		if ok {
			assert.Equal(t, tt.want, got, tt.dir)
		}
	}
}

func TestSSHCommandQuotesTheRemoteFolder(t *testing.T) {
	got := sshCommand(remote{host: "box", user: "me", port: 2222, dir: "/srv/it's here"})

	assert.Equal(t, "\x05\x15 ssh -t -p 2222 -- me@box 'cd '\\''/srv/it'\\''\\'\\'''\\''s here'\\''; exec \"${SHELL:-/bin/sh}\" -l'\r", got)
}

// Folders under a mount that exists, as gvfs and sshfs lay them out.
func mountedFolder(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.MkdirAll(path, 0o755))
	return path
}

func fakeSSH(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\necho \"ssh-args: $*\"\nprintf '\\033[?2004h'\nwhile read line; do [ \"$line\" = exit ] && exit 0; echo \"remote-got: $line\"; done\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ssh"), []byte(script), 0o755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
}

func TestSessionFollowsASFTPFolderOverSSH(t *testing.T) {
	fakeSSH(t)
	session, _ := startShell(t, &published{})
	require.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)
	root := t.TempDir()
	here := filepath.Join(root, "sftp:host=box,user=me", "srv")
	require.NoError(t, os.MkdirAll(filepath.Join(here, "logs"), 0o755))

	assert.True(t, session.FollowDirectory(here))
	require.Eventually(t, func() bool { return strings.Contains(screenText(session), "ssh-args: -t -- me@box") }, 5*time.Second, 20*time.Millisecond, screenText(session))
	assert.False(t, session.FollowDirectory(here), "already there")
	require.Eventually(t, func() bool { return !session.shellIdle() }, 5*time.Second, 20*time.Millisecond)

	other := filepath.Join(here, "logs")
	require.Eventually(t, func() bool { return session.FollowDirectory(other) }, 5*time.Second, 50*time.Millisecond)
	assert.Eventually(t, func() bool { return strings.Contains(screenText(session), "remote-got: cd -- '/srv/logs'") }, 5*time.Second, 20*time.Millisecond, screenText(session))
}

func TestSessionLeavesTheRemoteShellWhenTheFolderIsLocal(t *testing.T) {
	fakeSSH(t)
	session, dir := startShell(t, &published{})
	require.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)
	require.True(t, session.FollowDirectory(mountedFolder(t, "sftp:host=box,user=me/srv")))
	require.Eventually(t, func() bool { return !session.shellIdle() && strings.Contains(screenText(session), "ssh-args") }, 5*time.Second, 20*time.Millisecond)
	local := filepath.Join(dir, "local")
	require.NoError(t, os.Mkdir(local, 0o755))

	require.Eventually(t, func() bool { return session.FollowDirectory(local) }, 5*time.Second, 50*time.Millisecond)

	assert.Eventually(t, func() bool { return session.shellIdle() && session.shellCwd() == local }, 5*time.Second, 20*time.Millisecond, screenText(session))
}

func TestSessionMovesToAnotherServerThroughTheLocalShell(t *testing.T) {
	fakeSSH(t)
	session, _ := startShell(t, &published{})
	require.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)
	require.True(t, session.FollowDirectory(mountedFolder(t, "sftp:host=box,user=me/srv")))
	require.Eventually(t, func() bool { return !session.shellIdle() && strings.Contains(screenText(session), "ssh-args") }, 5*time.Second, 20*time.Millisecond)

	require.Eventually(t, func() bool { return session.FollowDirectory(mountedFolder(t, "sftp:host=other/etc")) }, 5*time.Second, 50*time.Millisecond)

	assert.Eventually(t, func() bool { return strings.Contains(screenText(session), "-- other") }, 5*time.Second, 20*time.Millisecond, screenText(session))
}

func TestElevatedLoginsRunTheirShellAsTheOtherUser(t *testing.T) {
	tests := []struct {
		as   sshfs.Elevation
		want string
	}{
		{sshfs.Elevation{}, `cd '/srv'; exec "${SHELL:-/bin/sh}" -l`},
		{sshfs.Elevation{Kind: "sudo"}, `cd '/srv'; exec sudo -s`},
		{sshfs.Elevation{Kind: "sudo", User: "nix"}, `cd '/srv'; exec sudo -s -u nix`},
		{sshfs.Elevation{Kind: "machinectl", User: "nix"}, `exec sudo machinectl shell -q nix@ /bin/sh -c 'cd '\''/srv'\'' 2>/dev/null; exec "${SHELL:-/bin/sh}" -l'`},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, remote{host: "box", dir: "/srv", as: tt.as}.shellScript(), tt.as.Tag())
	}
}

func TestElevatedAndPlainViewsOfAServerAreDifferentLogins(t *testing.T) {
	plain, _ := parseRemote("/x/sftp:host=box,user=me/srv")
	root, _ := parseRemote("/x/sftp:host=box,user=me,as=sudo/srv")

	assert.NotEqual(t, plain.key(), root.key())
}

func TestSessionAppliesTheElevationOfAServerWhoseFilesOpenAsYou(t *testing.T) {
	session := &Session{elevation: func(user, host string, port int) sshfs.Elevation {
		if user == "me" && host == "box" && port == 0 {
			return sshfs.Elevation{Kind: "machinectl", User: "nix"}
		}
		return sshfs.Elevation{}
	}}

	plain, _ := session.remoteAt("/x/sftp:host=box,user=me/srv")
	other, _ := session.remoteAt("/x/sftp:host=other,user=me/srv")
	own, _ := session.remoteAt("/x/sftp:host=box,user=me,as=sudo/srv")

	assert.Equal(t, "machinectl+nix", plain.as.Tag())
	assert.False(t, other.as.Active())
	assert.Equal(t, "sudo", own.as.Tag(), "a folder that carries its own elevation keeps it")
}

func TestSessionIgnoresAServerFolderThatIsNotMounted(t *testing.T) {
	fakeSSH(t)
	session, _ := startShell(t, &published{})
	require.Eventually(t, session.shellIdle, 5*time.Second, 20*time.Millisecond)

	assert.False(t, session.FollowDirectory("/run/user/1000/dfiles/sshfs/sftp:host=box,as=sudo/home/me"))
}
