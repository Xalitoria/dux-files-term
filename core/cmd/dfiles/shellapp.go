package main

import (
	"context"

	"github.com/AvengeMedia/dankfiles/core/internal/shellembed"
	"github.com/AvengeMedia/dankgo/shellapp"
)

var shellApp = shellapp.New(shellapp.Config{
	ID:       "dfiles",
	QSAppID:  "com.danklinux.dankfiles",
	Version:  Version,
	Embedded: embeddedShell{},
	Boot:     bootBackend,
})

func bootBackend(ctx context.Context) (shellapp.Backend, error) {
	svc, err := bootDaemonServices(ctx)
	if err != nil {
		return nil, err
	}
	return svc, nil
}

type embeddedShell struct{}

func (embeddedShell) Available() bool { return shellembed.Available() }

func (embeddedShell) Extract(baseDir string) (string, error) { return shellembed.Extract(baseDir) }

func (embeddedShell) Prune(baseDir, keep string) { shellembed.Prune(baseDir, keep) }
