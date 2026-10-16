package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/AvengeMedia/dankfiles/core/internal/bookmarks"
	"github.com/AvengeMedia/dankfiles/core/internal/clipboard"
	"github.com/AvengeMedia/dankfiles/core/internal/colorscheme"
	"github.com/AvengeMedia/dankfiles/core/internal/devices"
	"github.com/AvengeMedia/dankfiles/core/internal/elevate"
	"github.com/AvengeMedia/dankfiles/core/internal/filemanager1"
	"github.com/AvengeMedia/dankfiles/core/internal/fileops"
	"github.com/AvengeMedia/dankfiles/core/internal/ipc"
	"github.com/AvengeMedia/dankfiles/core/internal/mimeapps"
	"github.com/AvengeMedia/dankfiles/core/internal/netmount"
	"github.com/AvengeMedia/dankfiles/core/internal/paths"
	"github.com/AvengeMedia/dankfiles/core/internal/sandbox"
	"github.com/AvengeMedia/dankfiles/core/internal/search"
	"github.com/AvengeMedia/dankfiles/core/internal/settings"
	"github.com/AvengeMedia/dankgo/files"
	"github.com/AvengeMedia/dankgo/log"
	"github.com/godbus/dbus/v5"
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
	search    *search.Service
	bookmarks *bookmarks.Store
	fm1       *filemanager1.Service
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
	if s.fm1 != nil {
		s.fm1.Close()
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
	if s.search != nil {
		s.search.Close()
	}
	if s.bookmarks != nil {
		s.bookmarks.Close()
	}
	if s.ipc != nil {
		s.ipc.Close()
	}
}

func bootDaemonServices(ctx context.Context, ownsWindow bool) (*daemonServices, error) {
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

	config := configHome()
	networkService := netmount.NewService(bus, config)
	if err := networkService.Start(); err != nil {
		log.Warnf("gvfs unavailable: %v", err)
	}

	clipboardService := clipboard.NewService()
	opsManager.SetImageSource(clipboardService)
	searchService := search.NewService(bus, fileService)
	searchService.SetIndexEnabled(!caps.Sandboxed)
	searchService.SetMounts(func() []string { return localMounts(deviceService.Snapshot()) })
	searchService.SetExcludes(func() []string { return settings.Load(config).SearchExcludes })
	bookmarkStore := bookmarks.New(bus, config, filepath.Join(filepath.Dir(config), "gtk-3.0", "bookmarks"))
	if err := bookmarkStore.Start(); err != nil {
		log.Warnf("bookmarks unavailable: %v", err)
		bookmarkStore = nil
	}

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
		Search:      searchService,
		Bookmarks:   bookmarkStore,
		Settings:    func() settings.UISettings { return settings.Load(config) },
		Sandbox:     caps,
	}

	var fm1 *filemanager1.Service
	if ownsWindow {
		fm1 = startFileManager1(deps, dbusActivated || settings.Load(config).RegisterFileManager)
	}
	if fm1 != nil {
		deps.FileManager = fm1
	}

	ipcSrv, ipcErrCh, err := startIPC(ctx, deps)
	if err != nil {
		if fm1 != nil {
			fm1.Close()
		}
		return nil, err
	}

	return &daemonServices{ipc: ipcSrv, ipcErrCh: ipcErrCh, files: fileService, devices: deviceService, network: networkService, clipboard: clipboardService, search: searchService, bookmarks: bookmarkStore, fm1: fm1}, nil
}

// localMounts lists where the mounted local drives sit, leaving out the
// system device so scope Everywhere never walks the whole root filesystem.
func localMounts(snapshot devices.Snapshot) []string {
	var out []string
	for _, device := range snapshot.Devices {
		if !device.Mounted || device.System || device.Hidden {
			continue
		}
		if len(device.MountPoints) == 0 {
			out = append(out, device.MountPoint)
			continue
		}
		out = append(out, device.MountPoints...)
	}
	return out
}

type fileManagerRequests struct{ deps ipc.Deps }

func (r fileManagerRequests) ShowFolders(uris []string) error {
	return ipc.OpenLocations(r.deps, uris, nil, false)
}

func (r fileManagerRequests) ShowItems(uris []string) error {
	return ipc.OpenLocations(r.deps, nil, uris, false)
}

func (r fileManagerRequests) ShowItemProperties(uris []string) error {
	return ipc.ShowProperties(r.deps, uris)
}

func startFileManager1(deps ipc.Deps, enabled bool) *filemanager1.Service {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Warnf("filemanager1 unavailable: %v", err)
		return nil
	}
	svc, err := filemanager1.New(conn, fileManagerRequests{deps: deps})
	if err != nil {
		conn.Close()
		log.Warnf("filemanager1 unavailable: %v", err)
		return nil
	}
	svc.SetEnabled(enabled)
	return svc
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

	svc, err := bootDaemonServices(ctx, false)
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
