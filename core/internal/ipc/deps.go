package ipc

import (
	"github.com/AvengeMedia/dankfiles/core/internal/colorscheme"
	"github.com/AvengeMedia/dankfiles/core/internal/fileops"
	"github.com/AvengeMedia/dankfiles/core/internal/files"
)

type URIOpener interface {
	OpenURI(uri string) error
}

type Deps struct {
	Bus         *EventBus
	Pending     *PendingOpen
	Version     string
	ColorScheme *colorscheme.Watcher
	Opener      URIOpener
	Files       *files.Service
	FileOps     *fileops.Manager
}
