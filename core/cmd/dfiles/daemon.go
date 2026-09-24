package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankfiles/core/internal/clipboard"
	"github.com/AvengeMedia/dankfiles/core/internal/colorscheme"
	"github.com/AvengeMedia/dankfiles/core/internal/devices"
	"github.com/AvengeMedia/dankfiles/core/internal/elevate"
	"github.com/AvengeMedia/dankfiles/core/internal/fileops"
	"github.com/AvengeMedia/dankfiles/core/internal/ipc"
	"github.com/AvengeMedia/dankfiles/core/internal/mimeapps"
	"github.com/AvengeMedia/dankfiles/core/internal/netmount"
	"github.com/AvengeMedia/dankfiles/core/internal/paths"
	"github.com/AvengeMedia/dankfiles/core/internal/sandbox"
	"github.com/AvengeMedia/dankgo/files"
	"github.com/AvengeMedia/dankgo/log"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the dfiles daemon (IPC only, no UI)",
	RunE:  runDaemon,
}

type daemonServices struct {
	ipc       *ipc.Server
	ipcErrCh  <-chan error
	files     *files.Service
	devices   *devices.Service
	network   *netmount.Service
	clipboard *clipboard.Service
}

func (s *daemonServices) SocketPath() string {
	if s == nil || s.ipc == nil {
		return ""
	}
	return s.ipc.SocketPath()
}

func (s *daemonServices) Close() {
	if s == nil {
		return
	}
	if s.files != nil {
		s.files.Close()
	}
	if s.devices != nil {
		s.devices.Close()
	}
	if s.network != nil {
		s.network.Close()
	}
	if s.clipboard != nil {
		s.clipboard.Close()
	}
	if s.ipc != nil {
		s.ipc.Close()
	}
}

func bootDaemonServices(ctx context.Context) (*daemonServices, error) {
	caps := sandbox.Current()
	if caps.Sandboxed {
		log.Infof("sandboxed as %s (host filesystem: %v)", caps.AppID, caps.HostFilesystem)
	}
	sandbox.ExportHostDirs(caps)

	bus := ipc.NewEventBus()

	appService := mimeapps.NewService()
	fileService := files.NewService(bus, cacheHome(), appService.Detector())
	opsManager := fileops.NewManager(bus, fileService)
	opsManager.SetElevator(elevate.NewClient())

	deviceService := devices.NewService(bus, uint32(os.Getuid()))
	if err := deviceService.Start(); err != nil {
		log.Warnf("udisks2 unavailable: %v", err)
	}

	networkService := netmount.NewService(bus, configHome())
	if err := networkService.Start(); err != nil {
		log.Warnf("gvfs unavailable: %v", err)
	}

	clipboardService := clipboard.NewService()

	colorSchemeWatcher := colorscheme.New(bus.Publish)
	if err := colorSchemeWatcher.Start(); err != nil {
		log.Warnf("color-scheme portal watcher unavailable: %v", err)
	}

	deps := ipc.Deps{
		Bus:         bus,
		Pending:     &ipc.PendingOpen{},
		Version:     Version,
		ColorScheme: colorSchemeWatcher,
		Files:       fileService,
		FileOps:     opsManager,
		Apps:        appService,
		Devices:     deviceService,
		Network:     networkService,
		Clipboard:   clipboardService,
		Sandbox:     caps,
	}
	ipcSrv, ipcErrCh, err := startIPC(ctx, deps)
	if err != nil {
		return nil, err
	}

	return &daemonServices{ipc: ipcSrv, ipcErrCh: ipcErrCh, files: fileService, devices: deviceService, network: networkService, clipboard: clipboardService}, nil
}

func configHome() string {
	dir, err := paths.ConfigDir()
	if err != nil {
		log.Warnf("config dir unavailable: %v", err)
		return os.TempDir()
	}
	return dir
}

func cacheHome() string {
	dir, err := paths.CacheDir()
	if err != nil {
		log.Warnf("cache dir unavailable: %v", err)
		return os.TempDir()
	}
	return filepath.Dir(dir)
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
