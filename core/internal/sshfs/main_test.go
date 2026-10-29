package sshfs

import (
	"os"
	"testing"
)

// The password wrapper starts the test binary again as the relay, like it starts dfiles.
func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == "sshrelay" {
		os.Exit(Relay(os.Args[2], os.Args[3:], os.Getenv(PasswordEnv), PollableStdin(), os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}
