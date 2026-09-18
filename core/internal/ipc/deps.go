package ipc

import "github.com/AvengeMedia/dankfiles/core/internal/colorscheme"

type URIOpener interface {
	OpenURI(uri string) error
}

type Deps struct {
	Bus         *EventBus
	Pending     *PendingOpen
	Version     string
	ColorScheme *colorscheme.Watcher
	Opener      URIOpener
}
