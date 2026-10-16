package main

import (
	"os"

	"github.com/AvengeMedia/dankfiles/core/internal/devices"
	"github.com/AvengeMedia/dankfiles/core/internal/fileops"
	"github.com/AvengeMedia/dankfiles/core/internal/ipc"
	"github.com/AvengeMedia/dankfiles/core/internal/mimeapps"
	"github.com/AvengeMedia/dankfiles/core/internal/sandbox"
	"github.com/AvengeMedia/dankfiles/core/internal/search"
	"github.com/AvengeMedia/dankfiles/core/internal/settings"
	"github.com/AvengeMedia/dankgo/files"
	"github.com/AvengeMedia/dankgo/log"
)

func noServices(*ipc.EventBus) (ipc.Deps, func()) { return ipc.Deps{}, func() {} }

func appServices(*ipc.EventBus) (ipc.Deps, func()) {
	return ipc.Deps{Apps: mimeapps.NewService()}, func() {}
}

func fileServices(bus *ipc.EventBus) (ipc.Deps, func()) {
	apps := mimeapps.NewService()
	fileService := files.NewService(bus, cacheHome(), apps.Detector())
	return ipc.Deps{Apps: apps, Files: fileService, FileOps: fileops.NewManager(bus, fileService)}, fileService.Close
}

func searchServices(bus *ipc.EventBus) (ipc.Deps, func()) {
	fileService := files.NewService(bus, cacheHome(), mimeapps.NewService().Detector())
	searchService := search.NewService(bus, fileService)
	searchService.SetIndexEnabled(!sandbox.Current().Sandboxed)
	searchService.SetExcludes(func() []string { return settings.Load(configHome()).SearchExcludes })
	return ipc.Deps{Files: fileService, Search: searchService}, func() {
		searchService.Close()
		fileService.Close()
	}
}

func deviceServices(bus *ipc.EventBus) (ipc.Deps, func()) {
	deviceService := devices.NewService(bus, uint32(os.Getuid()))
	if err := deviceService.Start(); err != nil {
		log.Debugf("udisks2 unavailable: %v", err)
	}
	return ipc.Deps{Devices: deviceService}, deviceService.Close
}
