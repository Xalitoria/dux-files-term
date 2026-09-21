package sandbox

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/godbus/dbus/v5"
)

// Probe is everything Detect asks the environment; tests supply a fake.
type Probe interface {
	Getenv(key string) string
	ReadFile(path string) ([]byte, error)
	Exists(path string) bool
	LookPath(name string) (string, error)
	PortalVersion(dest, objectPath, iface string) (uint32, bool)
}

const portalProbeBudget = 2 * time.Second

// osProbe keeps one private bus connection for the portal lookups and
// closes it once Detect is done, so nothing lingers in the daemon.
type osProbe struct {
	conn     *dbus.Conn
	deadline time.Time
}

func newOSProbe() *osProbe { return &osProbe{deadline: time.Now().Add(portalProbeBudget)} }

func (p *osProbe) close() {
	if p.conn != nil {
		p.conn.Close()
	}
}

func (*osProbe) Getenv(key string) string { return os.Getenv(key) }

func (*osProbe) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (*osProbe) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (*osProbe) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (p *osProbe) PortalVersion(dest, objectPath, iface string) (uint32, bool) {
	if p.conn == nil {
		conn, err := dbus.ConnectSessionBus()
		if err != nil {
			return 0, false
		}
		p.conn = conn
	}
	ctx, cancel := context.WithDeadline(context.Background(), p.deadline)
	defer cancel()

	var version dbus.Variant
	err := p.conn.Object(dest, dbus.ObjectPath(objectPath)).
		CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", dbus.FlagNoAutoStart, iface, "version").
		Store(&version)
	if err != nil {
		return 0, false
	}
	v, ok := version.Value().(uint32)
	return v, ok
}
