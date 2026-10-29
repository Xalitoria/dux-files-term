package sshfs

import (
	"errors"
	"regexp"
	"strings"
)

const (
	KindSudo       = "sudo"
	KindMachinectl = "machinectl"
)

var userName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9._-]{0,31}$`)

// Elevation says who to become after logging in as a regular user. Files are always reached through sudo,
// machinectl only changes how the terminal gets there.
type Elevation struct {
	Kind string
	User string
}

func NewElevation(kind, user string) (Elevation, error) {
	user = strings.TrimSpace(user)
	if user == "root" {
		user = ""
	}
	switch {
	case kind == "":
		return Elevation{}, nil
	case kind != KindSudo && kind != KindMachinectl:
		return Elevation{}, errors.New("elevate must be sudo or machinectl")
	case kind == KindMachinectl && user == "":
		return Elevation{}, errors.New("machinectl needs a user to switch to")
	case user != "" && !userName.MatchString(user):
		return Elevation{}, errors.New("not a valid user name: " + user)
	}
	return Elevation{Kind: kind, User: user}, nil
}

func (e Elevation) Active() bool { return e.Kind != "" }

func (e Elevation) Target() string {
	if e.User == "" {
		return "root"
	}
	return e.User
}

// Tag is the short form used in folder names and the recent servers: "sudo", "sudo+nix", "machinectl+nix".
func (e Elevation) Tag() string {
	switch {
	case !e.Active():
		return ""
	case e.User == "":
		return e.Kind
	}
	return e.Kind + "+" + e.User
}

func ElevationFromTag(tag string) (Elevation, bool) {
	kind, user, _ := strings.Cut(tag, "+")
	elevation, err := NewElevation(kind, user)
	return elevation, err == nil && elevation.Active()
}

func (e Elevation) userArgs() string {
	if e.User == "" {
		return ""
	}
	return "-u " + e.User
}

func (e Elevation) sudoArgs() string {
	return strings.TrimSpace("-n " + e.userArgs())
}

// The command sshfs runs on the server in place of the sftp subsystem. With a password, the first line it
// receives is the password: sudo takes it if it asks, and a sudo that doesn't need one has it dropped.
func (e Elevation) ServerCommand(server string, password bool) string {
	if !password {
		return "sudo " + e.sudoArgs() + " " + quote(server)
	}
	script := `if sudo ` + e.sudoArgs() + ` true 2>/dev/null; then IFS= read -r _; exec sudo ` + e.sudoArgs() + ` "$0"; fi; ` +
		strings.TrimSpace(`exec sudo -S -p "" `+e.userArgs()) + ` "$0"`
	return "sh -c " + quote(script) + " " + quote(server)
}

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
