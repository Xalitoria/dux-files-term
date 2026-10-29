package sshfs

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ssh makes the socket it shares with sshfs non-blocking, so reads from it have to go through Go's poller.
func PollableStdin() *os.File {
	_ = syscall.SetNonblock(int(os.Stdin.Fd()), true)
	return os.NewFile(os.Stdin.Fd(), "stdin")
}

// Relay runs ssh with the sudo password in front of what sshfs sends. sshfs hands ssh a non-blocking
// socket, which cat and dd give up on, so the copying has to be done by Go.
func Relay(ssh string, args []string, password string, stdin io.Reader, stdout, stderr io.Writer) int {
	cmd := exec.Command(ssh, args...)
	cmd.Stdin = io.MultiReader(strings.NewReader(password+"\n"), stdin)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.ExitCode()
	}
	return 255
}
