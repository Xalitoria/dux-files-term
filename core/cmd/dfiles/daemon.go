package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankfiles/core/internal/colorscheme"
	"github.com/AvengeMedia/dankfiles/core/internal/ipc"
	"github.com/AvengeMedia/dankfiles/core/internal/uriopen"
	"github.com/AvengeMedia/dankgo/log"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the dfiles daemon (IPC only, no UI)",
	RunE:  runDaemon,
}

type daemonServices struct {
	ipc      *ipc.Server
	ipcErrCh <-chan error
}

func (s *daemonServices) SocketPath() string {
	if s == nil || s.ipc == nil {
		return ""
	}
	return s.ipc.SocketPath()
}

func (s *daemonServices) Close() {
	if s == nil || s.ipc == nil {
		return
	}
	s.ipc.Close()
}

func bootDaemonServices(ctx context.Context) (*daemonServices, error) {
	bus := ipc.NewEventBus()

	colorSchemeWatcher := colorscheme.New(bus.Publish)
	if err := colorSchemeWatcher.Start(); err != nil {
		log.Warnf("color-scheme portal watcher unavailable: %v", err)
	}

	deps := ipc.Deps{
		Bus:         bus,
		Pending:     &ipc.PendingOpen{},
		Version:     Version,
		ColorScheme: colorSchemeWatcher,
		Opener:      uriopen.Opener{},
	}
	ipcSrv, ipcErrCh, err := startIPC(ctx, deps)
	if err != nil {
		return nil, err
	}

	return &daemonServices{ipc: ipcSrv, ipcErrCh: ipcErrCh}, nil
}

func runDaemon(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := bootDaemonServices(ctx)
	if err != nil {
		return err
	}
	defer svc.Close()

	log.Infof("dfiles daemon ready (ipc=%s)", svc.SocketPath())

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		log.Infof("received %s, shutting down", sig)
	case err := <-svc.ipcErrCh:
		if err != nil {
			return fmt.Errorf("ipc server: %w", err)
		}
	}
	return nil
}

func startIPC(ctx context.Context, deps ipc.Deps) (*ipc.Server, <-chan error, error) {
	srv := ipc.NewServer(deps)
	if err := srv.Listen(); err != nil {
		return nil, nil, err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ctx)
	}()
	return srv, errCh, nil
}
