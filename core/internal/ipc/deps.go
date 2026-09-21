package ipc

import (
	"github.com/AvengeMedia/dankfiles/core/internal/colorscheme"
	"github.com/AvengeMedia/dankfiles/core/internal/devices"
	"github.com/AvengeMedia/dankfiles/core/internal/fileops"
	"github.com/AvengeMedia/dankfiles/core/internal/files"
	"github.com/AvengeMedia/dankfiles/core/internal/mimeapps"
	"github.com/AvengeMedia/dankfiles/core/internal/netmount"
	"github.com/AvengeMedia/dankfiles/core/internal/sandbox"
)

type Deps struct {
	Bus         *EventBus
	Pending     *PendingOpen
	Version     string
	ColorScheme *colorscheme.Watcher
	Files       *files.Service
	FileOps     *fileops.Manager
	Apps        *mimeapps.Service
	Devices     *devices.Service
	Network     *netmount.Service
	Sandbox     sandbox.Capabilities
}
