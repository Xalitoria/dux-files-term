package terminal

import (
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/duxlabs-dev/dux-files/core/internal/sshfs"
)

var (
	remoteHost = regexp.MustCompile(`^[A-Za-z0-9:][A-Za-z0-9._:%-]*$`)
	remoteUser = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._@-]*$`)
)

// An sftp location gvfs shows as a folder, such as /run/user/1000/gvfs/sftp:host=box,user=me/srv.
type remote struct {
	host, user string
	port       int
	dir        string
	as         sshfs.Elevation
}

func (r remote) key() string {
	return r.user + "@" + r.host + ":" + strconv.Itoa(r.port) + "|" + r.as.Tag()
}

func parseRemote(dir string) (remote, bool) {
	segments := strings.Split(path.Clean(dir), "/")
	for i, segment := range segments {
		kind, fields, found := strings.Cut(segment, ":")
		if !found || (kind != "sftp" && kind != "ssh") || i == 0 || segments[i-1] == "" {
			continue
		}
		return remoteFrom(fields, "/"+strings.Join(segments[i+1:], "/"))
	}
	return remote{}, false
}

func remoteFrom(fields, dir string) (remote, bool) {
	r := remote{dir: dir}
	for _, field := range strings.Split(fields, ",") {
		name, value, _ := strings.Cut(field, "=")
		value, err := url.PathUnescape(value)
		if err != nil {
			return remote{}, false
		}
		switch name {
		case "host":
			r.host = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
		case "user":
			r.user = value
		case "port":
			r.port, _ = strconv.Atoi(value)
		case "as":
			elevation, ok := sshfs.ElevationFromTag(value)
			if !ok && value != "" {
				return remote{}, false
			}
			r.as = elevation
		}
	}
	valid := remoteHost.MatchString(r.host) && (r.user == "" || remoteUser.MatchString(r.user)) && r.port >= 0 && r.port < 65536
	return r, valid
}

// Starts a login shell in the folder on the other side, and ends back in the local shell when it exits.
func sshCommand(r remote) string {
	args := []string{"ssh", "-t"}
	if r.port > 0 {
		args = append(args, "-p", strconv.Itoa(r.port))
	}
	target := r.host
	if r.user != "" {
		target = r.user + "@" + r.host
	}
	script := r.shellScript()
	return "\x05\x15 " + strings.Join(append(args, "--", target, shellQuote(script)), " ") + "\r"
}

const leaveCommand = "\x05\x15 exit\r"

// What runs on the server in place of the login shell. sudo keeps the folder, machinectl starts a session of its own and only root may ask for one.
func (r remote) shellScript() string {
	login := `exec "${SHELL:-/bin/sh}" -l`
	enter := "cd " + shellQuote(r.dir) + "; "
	switch r.as.Kind {
	case sshfs.KindSudo:
		return enter + "exec sudo " + strings.TrimSpace("-s "+userFlag(r.as))
	case sshfs.KindMachinectl:
		return "exec sudo machinectl shell -q " + r.as.User + "@ /bin/sh -c " + shellQuote("cd "+shellQuote(r.dir)+" 2>/dev/null; "+login)
	}
	return enter + login
}

func userFlag(as sshfs.Elevation) string {
	if as.User == "" {
		return ""
	}
	return "-u " + as.User
}

func remoteCdCommand(dir string) string {
	return "\x05\x15 cd -- " + shellQuote(dir) + "\r"
}

func commandName(pid int) string {
	name, _ := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/comm")
	return strings.TrimSpace(string(name))
}
