package sshfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/duxlabs-dev/dux-go/log"
)

const (
	readyTimeout = 20 * time.Second
	readyPoll    = 100 * time.Millisecond
	probeTimeout = 30 * time.Second
	stderrLimit  = 2048

	PasswordEnv = "DFILES_SUDO_PASSWORD"

	exitNoServer      = 20
	exitNeedsPassword = 21
	exitWrongPassword = 22
)

// ErrNoServer means the server only has the built-in SFTP, which can't be run as another user.
var ErrNoServer = errors.New("the server only has the built-in SFTP, which can't run as another user")

var (
	serverCandidates = []string{
		"/usr/lib/ssh/sftp-server", "/usr/lib/openssh/sftp-server", "/usr/libexec/openssh/sftp-server",
		"/usr/libexec/sftp-server", "/usr/lib64/misc/sftp-server", "/usr/lib/misc/sftp-server",
		"/run/current-system/sw/libexec/sftp-server",
	}
	hostName = regexp.MustCompile(`^[A-Za-z0-9:][A-Za-z0-9._:%-]*$`)
)

type Login struct {
	User string
	Host string
	Port int
}

func (l Login) target() string {
	if l.User == "" {
		return l.Host
	}
	return l.User + "@" + l.Host
}

func (l Login) URI() string {
	authority := l.Host
	if strings.Contains(authority, ":") {
		authority = "[" + authority + "]"
	}
	if l.Port > 0 {
		authority += ":" + strconv.Itoa(l.Port)
	}
	if l.User != "" {
		authority = url.User(l.User).String() + "@" + authority
	}
	return "sftp://" + authority + "/"
}

// ParseURI reads sftp://user@host:port/path. The path is where to land, the mount itself is always the whole server.
func ParseURI(raw string) (Login, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "sftp" && parsed.Scheme != "ssh") || parsed.Hostname() == "" {
		return Login{}, "", fmt.Errorf("%s is not an sftp address", raw)
	}
	login := Login{User: parsed.User.Username(), Host: parsed.Hostname()}
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number <= 0 || number > 65535 {
			return Login{}, "", fmt.Errorf("bad port %s", port)
		}
		login.Port = number
	}
	if !hostName.MatchString(login.Host) || (login.User != "" && !userName.MatchString(login.User)) {
		return Login{}, "", fmt.Errorf("%s is not a valid login", raw)
	}
	return login, "/" + strings.Trim(parsed.Path, "/"), nil
}

type Mounted struct {
	ID    string
	Name  string
	URI   string
	Path  string
	Home  string
	Login Login
	As    Elevation
}

type mount struct {
	Mounted
	cmd     *exec.Cmd
	exited  <-chan struct{}
	cleanup sync.Once
}

type Manager struct {
	root    string
	changed func()

	mu     sync.Mutex
	mounts map[string]*mount
}

// root holds one folder per mount, named like gvfs names its own so the terminal can tell where a folder lives.
func NewManager(root string, changed func()) *Manager {
	m := &Manager{root: root, changed: changed, mounts: map[string]*mount{}}
	m.cleanStale()
	return m
}

func folderName(l Login, e Elevation) string {
	fields := []string{"host=" + url.PathEscape(l.Host)}
	if l.User != "" {
		fields = append(fields, "user="+url.PathEscape(l.User))
	}
	if l.Port > 0 {
		fields = append(fields, "port="+strconv.Itoa(l.Port))
	}
	return "sftp:" + strings.Join(append(fields, "as="+e.Tag()), ",")
}

func (m *Manager) cleanStale() {
	entries, _ := os.ReadDir(m.root)
	for _, entry := range entries {
		dir := filepath.Join(m.root, entry.Name())
		_ = unmount(dir, true)
		_ = os.Remove(dir)
	}
}

func (m *Manager) List() []Mounted {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := make([]Mounted, 0, len(m.mounts))
	for _, mounted := range m.mounts {
		list = append(list, mounted.Mounted)
	}
	return list
}

func (m *Manager) Mount(login Login, as Elevation, password string) (Mounted, error) {
	id := folderName(login, as)
	m.mu.Lock()
	if existing := m.mounts[id]; existing != nil {
		m.mu.Unlock()
		return existing.Mounted, nil
	}
	m.mu.Unlock()

	sshfs, ssh, err := tools()
	if err != nil {
		return Mounted{}, err
	}
	server, home, err := probe(ssh, login, as, password)
	if err != nil {
		return Mounted{}, err
	}
	dir := filepath.Join(m.root, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Mounted{}, err
	}
	cmd, stderr, err := startSSHFS(sshfs, ssh, dir, login, as, server, password)
	if err != nil {
		_ = os.Remove(dir)
		return Mounted{}, err
	}
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	if err := waitMounted(dir, exited, stderr); err != nil {
		_ = cmd.Process.Kill()
		<-exited
		_ = unmount(dir, true)
		_ = os.Remove(dir)
		return Mounted{}, err
	}

	mounted := &mount{Mounted: Mounted{
		ID: id, Name: nameOf(login, as), URI: login.URI(), Path: dir, Home: home, Login: login, As: as,
	}, cmd: cmd, exited: exited}
	m.mu.Lock()
	m.mounts[id] = mounted
	m.mu.Unlock()
	go func() {
		<-exited
		if m.Owns(id) {
			log.Warnf("sshfs: %s ended: %s", mounted.Name, lastLine(stderr.String(), "no message"))
		}
		m.forget(mounted)
	}()
	return mounted.Mounted, nil
}

func nameOf(l Login, e Elevation) string {
	return e.Target() + " on " + l.Host + " (" + e.Kind + ")"
}

func (m *Manager) forget(mounted *mount) {
	mounted.cleanup.Do(func() {
		m.mu.Lock()
		if m.mounts[mounted.ID] == mounted {
			delete(m.mounts, mounted.ID)
		}
		m.mu.Unlock()
		_ = unmount(mounted.Path, true)
		_ = os.Remove(mounted.Path)
		if m.changed != nil {
			m.changed()
		}
	})
}

func (m *Manager) Owns(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mounts[id] != nil
}

func (m *Manager) Unmount(id string) error {
	m.mu.Lock()
	mounted := m.mounts[id]
	m.mu.Unlock()
	if mounted == nil {
		return errors.New("not mounted: " + id)
	}
	if err := unmount(mounted.Path, false); err != nil {
		return err
	}
	_ = mounted.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-mounted.exited:
	case <-time.After(readyTimeout):
	}
	m.forget(mounted)
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	all := make([]*mount, 0, len(m.mounts))
	for _, mounted := range m.mounts {
		all = append(all, mounted)
	}
	m.mounts = map[string]*mount{}
	m.mu.Unlock()
	for _, mounted := range all {
		_ = unmount(mounted.Path, true)
		_ = mounted.cmd.Process.Signal(syscall.SIGTERM)
		_ = os.Remove(mounted.Path)
	}
}

func unmount(dir string, lazy bool) error {
	tool, err := exec.LookPath("fusermount3")
	if err != nil {
		tool, err = exec.LookPath("fusermount")
	}
	if err != nil {
		return err
	}
	args := []string{"-u", dir}
	if lazy {
		args = []string{"-uz", dir}
	}
	if out, err := exec.Command(tool, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func tools() (sshfs, ssh string, err error) {
	if sshfs, err = find("sshfs"); err != nil {
		return "", "", errors.New("sshfs is needed to open a server as another user, install it and fuse3")
	}
	if ssh, err = find("ssh"); err != nil {
		return "", "", errors.New("ssh is not installed")
	}
	return sshfs, ssh, nil
}

func find(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	home, _ := os.UserHomeDir()
	for _, dir := range []string{filepath.Join(home, ".nix-profile", "bin"), "/run/current-system/sw/bin", "/usr/local/bin", "/usr/bin"} {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}

func sshArgs(login Login, extra ...string) []string {
	args := []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10"}
	if login.Port > 0 {
		args = append(args, "-p", strconv.Itoa(login.Port))
	}
	return append(append(args, extra...), "--", login.target())
}

// Logs in once to find the server's sftp-server and to check sudo, so a wrong password or a server with only the
// built-in sftp is reported before anything is mounted.
func probe(ssh string, login Login, as Elevation, password string) (server, home string, err error) {
	script := fmt.Sprintf(`pw=%d; s=; for p in %s; do [ -x "$p" ] && s=$p && break; done
if [ -z "$s" ]; then d=$(dirname "$(readlink -f "$(command -v ssh)" 2>/dev/null)" 2>/dev/null)/../libexec/sftp-server; [ -x "$d" ] && s=$d; fi
[ -n "$s" ] || exit %d
echo "$s"
getent passwd %s | cut -d: -f6
sudo %s true 2>/dev/null && exit 0
[ "$pw" = 1 ] || exit %d
sudo -S -p "" %s true 2>/dev/null || exit %d`,
		boolInt(password != ""), strings.Join(serverCandidates, " "), exitNoServer, as.Target(),
		as.sudoArgs(), exitNeedsPassword, as.userArgs(), exitWrongPassword)

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, ssh, append(sshArgs(login), "sh -c "+quote(script))...)
	cmd.Stdin = strings.NewReader(password + "\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	var exit *exec.ExitError
	switch {
	case runErr == nil:
		lines := append(strings.Split(strings.TrimSpace(stdout.String()), "\n"), "")
		return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1]), nil
	case !errors.As(runErr, &exit):
		return "", "", runErr
	}
	switch exit.ExitCode() {
	case exitNoServer:
		return "", "", ErrNoServer
	case exitNeedsPassword:
		return "", "", errors.New("sudo asks for a password, enter it to connect")
	case exitWrongPassword:
		return "", "", errors.New("sudo did not accept the password")
	}
	return "", "", errors.New(lastLine(stderr.String(), "could not log in over ssh"))
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func lastLine(text, fallback string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if last := strings.TrimSpace(lines[len(lines)-1]); last != "" {
		return last
	}
	return fallback
}

type tail struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf.Write(p)
	if over := t.buf.Len() - stderrLimit; over > 0 {
		t.buf.Next(over)
	}
	return len(p), nil
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

func startSSHFS(sshfs, ssh, dir string, login Login, as Elevation, server, password string) (*exec.Cmd, *tail, error) {
	options := []string{
		"sftp_server=" + as.ServerCommand(server, password != ""),
		"reconnect", "ServerAliveInterval=15", "ServerAliveCountMax=3", "BatchMode=yes", "ConnectTimeout=10",
		"fsname=dfiles-sshfs",
	}
	if login.Port > 0 {
		options = append(options, "port="+strconv.Itoa(login.Port))
	}
	env := os.Environ()
	command := ssh
	if password != "" {
		wrapper, err := writeWrapper(filepath.Dir(dir), ssh)
		if err != nil {
			return nil, nil, err
		}
		command = wrapper
		env = append(env, PasswordEnv+"="+password)
	}
	options = append(options, "ssh_command="+command)
	cmd := exec.Command(sshfs, "-f", "-o", strings.Join(options, ","), login.target()+":/", dir)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stderr := &tail{}
	cmd.Stderr = stderr
	return cmd, stderr, cmd.Start()
}

// Puts the password in front of the sftp traffic, where the server command's sudo reads it.
func writeWrapper(root, ssh string) (string, error) {
	path := filepath.Join(root, ".ssh-with-password")
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	script := "#!/bin/sh\nexec " + quote(self) + " sshrelay " + quote(ssh) + ` "$@"` + "\n"
	return path, os.WriteFile(path, []byte(script), 0o700)
}

func waitMounted(dir string, exited <-chan struct{}, stderr *tail) error {
	deadline := time.After(readyTimeout)
	for {
		select {
		case <-exited:
			return errors.New(lastLine(stderr.String(), "sshfs stopped before the server was mounted"))
		case <-deadline:
			return errors.New(lastLine(stderr.String(), "timed out mounting the server"))
		case <-time.After(readyPoll):
		}
		if isMountpoint(dir) {
			return nil
		}
	}
}

func isMountpoint(dir string) bool {
	table, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(table), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 4 && unescapeMount(fields[4]) == dir {
			return true
		}
	}
	return false
}

// mountinfo writes spaces, tabs, newlines and backslashes as octal escapes.
func unescapeMount(path string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(path)
}
