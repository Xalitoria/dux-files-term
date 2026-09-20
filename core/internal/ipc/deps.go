package ipc

import (
	"github.com/AvengeMedia/dankfiles/core/internal/colorscheme"
	"github.com/AvengeMedia/dankfiles/core/internal/fileops"
	"github.com/AvengeMedia/dankfiles/core/internal/files"
	"github.com/AvengeMedia/dankfiles/core/internal/mimeapps"
)

type Deps struct {
	Bus         *EventBus
	Pending     *PendingOpen
	Version     string
	ColorScheme *colorscheme.Watcher
	Files       *files.Service
	FileOps     *fileops.Manager
	Apps        *mimeapps.Service
}
