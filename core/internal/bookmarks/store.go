package bookmarks

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/AvengeMedia/dankfiles/core/internal/atomicfile"
	"github.com/AvengeMedia/dankgo/log"
)

const (
	FileName       = "bookmarks.json"
	Topic          = "bookmarks"
	coalesceWindow = 50 * time.Millisecond
)

type Publisher interface {
	Publish(topic string, data any)
}

type Bookmark struct {
	Path  string `json:"path"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

type changed struct {
	Kind      string     `json:"kind"`
	Bookmarks []Bookmark `json:"bookmarks"`
}

type Store struct {
	bus     Publisher
	dir     string
	file    string
	gtkFile string

	mu          sync.Mutex
	items       []Bookmark
	lastWritten []byte
	gtkImport   bool

	watcher *fsnotify.Watcher
	done    chan struct{}
	stopped sync.WaitGroup
}

func New(bus Publisher, configDir, gtkFile string) *Store {
	return &Store{bus: bus, dir: configDir, file: filepath.Join(configDir, FileName), gtkFile: gtkFile}
}

// The directory is watched, not the file: other editors replace it by rename.
func (s *Store) Start() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	s.mu.Lock()
	s.items, s.lastWritten, _ = s.load()
	s.gtkImport = s.gtkImportAvailable()
	s.mu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(s.dir); err != nil {
		_ = watcher.Close()
		return err
	}
	s.watcher = watcher
	s.done = make(chan struct{})
	s.stopped.Add(1)
	go s.watch()
	return nil
}

func (s *Store) Close() {
	if s.watcher == nil {
		return
	}
	close(s.done)
	_ = s.watcher.Close()
	s.stopped.Wait()
	s.watcher = nil
}

func (s *Store) List() []Bookmark {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.items)
}

func (s *Store) Exists() bool {
	_, err := os.Stat(s.file)
	return err == nil
}

func (s *Store) Add(b Bookmark) ([]Bookmark, error) {
	if !filepath.IsAbs(b.Path) {
		return nil, errors.New("bookmark path must be absolute")
	}
	b.Path = filepath.Clean(b.Path)
	if b.Label == "" {
		b.Label = labelOf(b.Path)
	}
	return s.update(func(items []Bookmark) ([]Bookmark, error) {
		if indexOf(items, b.Path) >= 0 {
			return items, nil
		}
		return append(items, b), nil
	})
}

func (s *Store) Remove(path string) ([]Bookmark, error) {
	return s.update(func(items []Bookmark) ([]Bookmark, error) {
		i := indexOf(items, path)
		if i < 0 {
			return items, nil
		}
		return slices.Delete(items, i, i+1), nil
	})
}

func (s *Store) Rename(path, label string) ([]Bookmark, error) {
	return s.update(func(items []Bookmark) ([]Bookmark, error) {
		i := indexOf(items, path)
		if i < 0 {
			return nil, errors.New("not a bookmark: " + path)
		}
		if label == "" {
			label = labelOf(path)
		}
		items[i].Label = label
		return items, nil
	})
}

func (s *Store) Reorder(paths []string) ([]Bookmark, error) {
	return s.update(func(items []Bookmark) ([]Bookmark, error) {
		if len(paths) != len(items) {
			return nil, errors.New("reorder must list every bookmark")
		}
		next := make([]Bookmark, 0, len(items))
		for _, path := range paths {
			i := indexOf(items, path)
			if i < 0 || slices.ContainsFunc(next, func(b Bookmark) bool { return b.Path == path }) {
				return nil, errors.New("reorder must list every bookmark once")
			}
			next = append(next, items[i])
		}
		return next, nil
	})
}

func (s *Store) update(apply func([]Bookmark) ([]Bookmark, error)) ([]Bookmark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next, err := apply(slices.Clone(s.items))
	if err != nil {
		return nil, err
	}
	if err := s.write(next); err != nil {
		return nil, err
	}
	s.items = next
	s.publish(next)
	return slices.Clone(next), nil
}

func (s *Store) write(items []Bookmark) error {
	data, err := json.MarshalIndent(items, "", " ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := atomicfile.WriteFile(s.file, data, 0o644); err != nil {
		return err
	}
	s.lastWritten = data
	s.gtkImport = false
	return nil
}

func (s *Store) load() ([]Bookmark, []byte, bool) {
	data, err := os.ReadFile(s.file)
	if err != nil {
		return []Bookmark{}, nil, true
	}
	var items []Bookmark
	if err := json.Unmarshal(data, &items); err != nil {
		log.Warnf("bookmarks: ignoring unreadable %s: %v", s.file, err)
		return []Bookmark{}, data, false
	}
	kept := items[:0]
	for _, b := range items {
		if !filepath.IsAbs(b.Path) {
			continue
		}
		if b.Label == "" {
			b.Label = labelOf(b.Path)
		}
		kept = append(kept, b)
	}
	return kept, data, true
}

func (s *Store) watch() {
	defer s.stopped.Done()
	reload := time.NewTimer(coalesceWindow)
	if !reload.Stop() {
		<-reload.C
	}
	armed := false

	for {
		select {
		case <-s.done:
			return
		case ev, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(ev.Name) != FileName || armed {
				continue
			}
			reload.Reset(coalesceWindow)
			armed = true
		case _, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
		case <-reload.C:
			armed = false
			s.reload()
		}
	}
}

func (s *Store) reload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, data, ok := s.load()
	s.gtkImport = s.gtkImportAvailable()
	if !ok || bytes.Equal(data, s.lastWritten) {
		return
	}
	s.items = items
	s.lastWritten = data
	s.publish(items)
}

func (s *Store) publish(items []Bookmark) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(Topic, changed{Kind: "changed", Bookmarks: slices.Clone(items)})
}

func indexOf(items []Bookmark, path string) int {
	return slices.IndexFunc(items, func(b Bookmark) bool { return b.Path == path })
}

func labelOf(path string) string {
	if path == "/" {
		return "/"
	}
	return filepath.Base(path)
}
