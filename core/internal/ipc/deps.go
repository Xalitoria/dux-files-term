package ipc

import (
	"github.com/AvengeMedia/dankgo/files"
	"github.com/duxlabsdev/dux-files/core/internal/bookmarks"
	"github.com/duxlabsdev/dux-files/core/internal/clipboard"
	"github.com/duxlabsdev/dux-files/core/internal/colorscheme"
	"github.com/duxlabsdev/dux-files/core/internal/devices"
	"github.com/duxlabsdev/dux-files/core/internal/filemanager1"
	"github.com/duxlabsdev/dux-files/core/internal/fileops"
	"github.com/duxlabsdev/dux-files/core/internal/mimeapps"
	"github.com/duxlabsdev/dux-files/core/internal/netmount"
	"github.com/duxlabsdev/dux-files/core/internal/portal"
	"github.com/duxlabsdev/dux-files/core/internal/recent"
	"github.com/duxlabsdev/dux-files/core/internal/sandbox"
	"github.com/duxlabsdev/dux-files/core/internal/search"
	"github.com/duxlabsdev/dux-files/core/internal/settings"
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
	Search      *search.Service
	Bookmarks   *bookmarks.Store
	Recent      *recent.Store
	Settings    func() settings.UISettings
	Sandbox     sandbox.Capabilities
	FileManager *filemanager1.Service
	Portal      *portal.Controller
	Quit        func()
}
