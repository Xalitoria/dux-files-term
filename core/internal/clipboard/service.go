package clipboard

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/AvengeMedia/dankgo/wlclipboard"
)

// selection is the wayland side of the clipboard, behind an interface so the
// tests never touch a compositor.
type selection interface {
	Set(offers []wlclipboard.Offer) error
	Close()
}

type reader func() ([]byte, string, error)

// Service owns the wayland selection while dfiles holds it. The owner keeps a
// wayland connection and a goroutine alive, so it is created on the first copy
// and dropped again on clear or shutdown.
type Service struct {
	newOwner func() (selection, error)
	paste    reader

	mu       sync.Mutex
	owner    selection
	paths    []string
	cut      bool
	plain    string
	unusable bool
}

func NewService() *Service {
	return &Service{
		newOwner: func() (selection, error) { return wlclipboard.NewOwner() },
		paste:    wlclipboard.Paste,
	}
}

func Handles(method string) bool { return strings.HasPrefix(method, "clipboard.") }

func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drop()
}

// Copy takes the selection for paths. cut marks them for a move.
func (s *Service) Copy(paths []string, cut bool) error {
	if len(paths) == 0 {
		return errors.New("clipboard.copy requires paths")
	}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			return errors.New("clipboard.copy requires absolute paths: " + path)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner == nil {
		owner, err := s.newOwner()
		if err != nil {
			s.unusable = true
			return err
		}
		s.owner = owner
	}

	plain := string(PlainText(paths))
	if err := s.owner.Set(offers(paths, cut, plain)); err != nil {
		s.drop()
		return err
	}
	s.paths = append([]string(nil), paths...)
	s.cut = cut
	s.plain = plain
	return nil
}

// Paths reads the selection back. Our own is returned exactly; anything else is
// parsed out of whatever flavour the owner serves.
func (s *Service) Paths() ([]string, bool, string) {
	data, mime, err := s.paste()
	if err != nil {
		return nil, false, ""
	}

	s.mu.Lock()
	mine := s.paths != nil && string(data) == s.plain
	paths, cut := append([]string(nil), s.paths...), s.cut
	s.mu.Unlock()

	if mine {
		return paths, cut, "dfiles"
	}
	foreign, foreignCut := Parse(data, mime)
	return foreign, foreignCut, mime
}

func (s *Service) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drop()
}

// Available reports whether the compositor let us take the selection. It stays
// true until a copy actually fails, so the UI does not probe wayland at startup.
func (s *Service) Available() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.unusable
}

func (s *Service) drop() {
	if s.owner != nil {
		s.owner.Close()
		s.owner = nil
	}
	s.paths = nil
	s.cut = false
	s.plain = ""
}

func offers(paths []string, cut bool, plain string) []wlclipboard.Offer {
	list := []wlclipboard.Offer{
		{MimeType: MimeGnomeCopiedFiles, Data: GnomePayload(paths, cut)},
		{MimeType: MimeUriList, Data: UriList(paths)},
	}
	if cut {
		list = append(list, wlclipboard.Offer{MimeType: MimeKdeCutSelection, Data: []byte("1")})
	}
	return append(list, wlclipboard.ExpandOffers([]byte(plain), MimeTextUTF8)...)
}
