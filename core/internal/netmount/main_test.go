package netmount

import (
	"os"
	"testing"
)

// A service clears the sshfs folders it finds, so tests must not look at the real ones.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "netmount-runtime")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_RUNTIME_DIR", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
