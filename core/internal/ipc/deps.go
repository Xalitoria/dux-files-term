package ipc

import (
	"github.com/AvengeMedia/dankfiles/core/internal/clipboard"
	"github.com/AvengeMedia/dankfiles/core/internal/colorscheme"
	"github.com/AvengeMedia/dankfiles/core/internal/devices"
	"github.com/AvengeMedia/dankfiles/core/internal/fileops"
	"github.com/AvengeMedia/dankfiles/core/internal/mimeapps"
	"github.com/AvengeMedia/dankfiles/core/internal/netmount"
	"github.com/AvengeMedia/dankfiles/core/internal/sandbox"
	"github.com/AvengeMedia/dankgo/files"
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
	Clipboard   *clipboard.Service
	Sandbox     sandbox.Capabilities
}
