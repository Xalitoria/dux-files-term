package files

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	pendingCap     = 64
	coalesceWindow = 100 * time.Millisecond
	enrichChunk    = 64
	thumbChunk     = 4
)

type notifier interface {
	Add(path string) error
	Events() <-chan fsnotify.Event
	Errors() <-chan error
	Close() error
}

type fsnotifyWatcher struct{ inner *fsnotify.Watcher }

func newFsnotifyWatcher() (notifier, error) {
	inner, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &fsnotifyWatcher{inner: inner}, nil
}

func (w *fsnotifyWatcher) Add(path string) error         { return w.inner.Add(path) }
func (w *fsnotifyWatcher) Events() <-chan fsnotify.Event { return w.inner.Events }
func (w *fsnotifyWatcher) Errors() <-chan error          { return w.inner.Errors }
func (w *fsnotifyWatcher) Close() error                  { return w.inner.Close() }

type batch struct {
	WatchID    string   `json:"watchId"`
	Kind       string   `json:"kind"`
	Seq        int      `json:"seq"`
	Added      []Entry  `json:"added,omitempty"`
	AddedAt    []int    `json:"addedAt,omitempty"`
	Changed    []Entry  `json:"changed,omitempty"`
	Removed    []string `json:"removed,omitempty"`
	Thumbnails []Entry  `json:"thumbnails,omitempty"`
}

type watch struct {
	id          string
	topic       string
	path        string
	svc         *Service
	notifier    notifier
	watching    bool
	pollOnFocus bool
	stopFromCtx func() bool
	done        chan struct{}
	closeOnce   sync.Once

	mu            sync.Mutex
	entries       []Entry
	spec          SortSpec
	includeHidden bool
	seq           int
	subscribed    bool
	pending       []batch
	dropped       bool
}

func (s *Service) openWatch(ctx context.Context, path string, spec SortSpec, includeHidden bool) (*watch, error) {
	entries, err := s.builder.readDir(path)
	if err != nil {
		return nil, err
	}
	sortEntries(entries, spec)

	w := &watch{
		path:          path,
		svc:           s,
		done:          make(chan struct{}),
		entries:       entries,
		spec:          spec,
		includeHidden: includeHidden,
		pollOnFocus:   needsFocusRefresh(path),
	}
	w.id = s.register(w)
	w.topic = "files:" + w.id

	notify, err := s.newNotifier()
	switch {
	case err != nil:
		s.log("watch %s: notifications unavailable: %v", path, err)
	default:
		if addErr := notify.Add(path); addErr != nil {
			s.log("watch %s: %v", path, addErr)
			_ = notify.Close()
			break
		}
		w.notifier = notify
		w.watching = true
		go w.run()
	}

	w.stopFromCtx = context.AfterFunc(ctx, w.close)
	w.enrich(entries)
	return w, nil
}

func (w *watch) run() {
	pending := map[string]bool{}
	flush := time.NewTimer(coalesceWindow)
	if !flush.Stop() {
		<-flush.C
	}
	armed := false

	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.notifier.Events():
			if !ok {
				return
			}
			if w.isSelf(event) {
				w.publishGone()
				return
			}
			pending[filepath.Base(event.Name)] = true
			if !armed {
				flush.Reset(coalesceWindow)
				armed = true
			}
		case err, ok := <-w.notifier.Errors():
			if !ok {
				return
			}
			if errors.Is(err, fsnotify.ErrEventOverflow) {
				w.resync()
				clear(pending)
				armed = false
			}
		case <-flush.C:
			armed = false
			names := slices.Collect(maps.Keys(pending))
			clear(pending)
			w.flush(names)
		}
	}
}

func (w *watch) isSelf(event fsnotify.Event) bool {
	if event.Name != w.path {
		return false
	}
	return event.Op&(fsnotify.Remove|fsnotify.Rename) != 0
}

func (w *watch) flush(names []string) {
	if _, err := os.Stat(w.path); err != nil {
		w.publishGone()
		return
	}

	hidden := hiddenNames(w.path)
	fresh := make([]Entry, 0, len(names))

	w.mu.Lock()
	before := w.positions()
	touched := map[string]bool{}
	for _, name := range names {
		touched[name] = true
		entry, err := w.svc.builder.entry(w.path, name, hidden[name])
		index := w.indexOf(name)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if index >= 0 {
				w.entries = slices.Delete(w.entries, index, index+1)
			}
		case err != nil:
			continue
		case index >= 0:
			entry.Thumbnail = carryThumbnail(w.entries[index], entry)
			w.entries[index] = entry
			w.entries = w.resortAround(index)
			fresh = append(fresh, entry)
		default:
			w.entries = insertSorted(w.entries, entry, w.spec)
			fresh = append(fresh, entry)
		}
	}
	w.publishLocked(w.diff(before, touched))
	w.mu.Unlock()

	w.enrich(fresh)
}

func (w *watch) positions() map[string]int {
	view := filterHidden(w.entries, w.includeHidden)
	out := make(map[string]int, len(view))
	for i, e := range view {
		out[e.Name] = i
	}
	return out
}

// diff turns the touched names into a batch the UI can apply without knowing
// the sort order: removals by name, then insertions at their final index.
func (w *watch) diff(before map[string]int, touched map[string]bool) batch {
	view := filterHidden(w.entries, w.includeHidden)
	changes := batch{Kind: "batch"}

	for name := range touched {
		_, existed := before[name]
		if existed && !containsName(view, name) {
			changes.Removed = append(changes.Removed, name)
		}
	}

	for index, entry := range view {
		if !touched[entry.Name] {
			continue
		}
		previous, existed := before[entry.Name]
		switch {
		case !existed:
			changes.Added = append(changes.Added, entry)
			changes.AddedAt = append(changes.AddedAt, index)
		case previous != index:
			changes.Removed = append(changes.Removed, entry.Name)
			changes.Added = append(changes.Added, entry)
			changes.AddedAt = append(changes.AddedAt, index)
		default:
			changes.Changed = append(changes.Changed, entry)
		}
	}
	return changes
}

func (w *watch) resortAround(index int) []Entry {
	entry := w.entries[index]
	rest := slices.Delete(slices.Clone(w.entries), index, index+1)
	return insertSorted(rest, entry, w.spec)
}

func (w *watch) indexOf(name string) int {
	return slices.IndexFunc(w.entries, func(e Entry) bool { return e.Name == name })
}

func carryThumbnail(old, fresh Entry) string {
	if old.MtimeMs == fresh.MtimeMs {
		return old.Thumbnail
	}
	return ""
}

func containsName(entries []Entry, name string) bool {
	return slices.ContainsFunc(entries, func(e Entry) bool { return e.Name == name })
}

func (w *watch) resync() {
	entries, err := w.svc.builder.readDir(w.path)
	if err != nil {
		w.publishGone()
		return
	}

	w.mu.Lock()
	sortEntries(entries, w.spec)
	w.entries = entries
	w.publishLocked(batch{Kind: "resync"})
	w.mu.Unlock()

	w.enrich(entries)
}

func (w *watch) enrich(entries []Entry) {
	pending := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Unreadable || e.IsDir || e.SymlinkBroken {
			continue
		}
		if !needsSniff(e.Mime) && !e.Thumbnailable && !needsDesktop(e) {
			continue
		}
		pending = append(pending, e)
	}

	for chunk := range slices.Chunk(pending, enrichChunk) {
		work := slices.Clone(chunk)
		w.svc.pool.submit(func() { w.enrichChunk(work) })
	}
}

func (w *watch) enrichChunk(chunk []Entry) {
	changed := make([]Entry, 0, len(chunk))
	for _, e := range chunk {
		if w.closed() {
			return
		}
		updated := e
		if needsSniff(updated.Mime) {
			if sniffed := w.svc.builder.mime.Sniff(updated.Path); sniffed != "" {
				updated.Mime = sniffed
			}
		}
		updated.Thumbnailable = w.svc.thumbs.Supports(updated.Mime)
		if updated.Thumbnailable {
			updated.Thumbnail, _ = w.svc.thumbs.Lookup(updated.Path, updated.MtimeMs, "normal")
		}
		updated.IconName = w.svc.builder.icons.best(updated)
		w.svc.builder.display(&updated)
		if updated.Mime == e.Mime && updated.IconName == e.IconName && updated.Thumbnail == e.Thumbnail && updated.Thumbnailable == e.Thumbnailable && updated.DisplayName == e.DisplayName {
			continue
		}
		changed = append(changed, updated)
	}
	w.applyChanged(changed, "batch")
}

func (w *watch) applyChanged(changed []Entry, kind string) {
	if len(changed) == 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	applied := make([]Entry, 0, len(changed))
	for _, e := range changed {
		index := w.indexOf(e.Name)
		if index < 0 || w.entries[index].MtimeMs != e.MtimeMs {
			continue
		}
		e.nameKey = w.entries[index].nameKey
		w.entries[index] = e
		applied = append(applied, e)
	}
	if !w.includeHidden {
		applied = visible(applied)
	}
	if len(applied) == 0 {
		return
	}

	switch kind {
	case "thumbnails":
		w.publishLocked(batch{Kind: kind, Thumbnails: applied})
	default:
		w.publishLocked(batch{Kind: kind, Changed: applied})
	}
}

// The watch is closed before the event goes out, so a client that has been told
// its directory is gone can never still resolve the watch id.
func (w *watch) publishGone() {
	w.close()
	w.mu.Lock()
	w.publishLocked(batch{Kind: "gone"})
	w.mu.Unlock()
}

func (w *watch) publishLocked(payload batch) {
	if payload.Kind == "batch" && len(payload.Added)+len(payload.Changed)+len(payload.Removed) == 0 {
		return
	}
	w.seq++
	payload.WatchID = w.id
	payload.Seq = w.seq

	if w.subscribed {
		w.svc.publish(w.topic, payload)
		return
	}
	if len(w.pending) >= pendingCap {
		w.pending = nil
		w.dropped = true
		return
	}
	w.pending = append(w.pending, payload)
}

// attach flushes what the watch produced between files.watch and the client
// subscribing to its topic; a client that never subscribes buffers nothing
// beyond pendingCap and gets one resync instead.
func (w *watch) attach() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.subscribed = true
	buffered := w.pending
	w.pending = nil

	if w.dropped {
		w.dropped = false
		w.seq++
		w.svc.publish(w.topic, batch{WatchID: w.id, Kind: "resync", Seq: w.seq})
		return
	}
	for _, payload := range buffered {
		w.svc.publish(w.topic, payload)
	}
}

func (w *watch) sequence() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.seq
}

func visible(entries []Entry) []Entry {
	return slices.DeleteFunc(slices.Clone(entries), func(e Entry) bool { return e.Hidden })
}

func (w *watch) page(opts ListOptions) Page {
	w.mu.Lock()
	defer w.mu.Unlock()

	if opts.Sort != w.spec {
		w.spec = opts.Sort
		sortEntries(w.entries, w.spec)
	}
	w.includeHidden = opts.IncludeHidden
	return pageOf(filterHidden(w.entries, opts.IncludeHidden), opts.Cursor, opts.Limit)
}

func (w *watch) entry(name string) (Entry, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	index := w.indexOf(name)
	if index < 0 {
		return Entry{}, false
	}
	return w.entries[index], true
}

func (w *watch) closed() bool {
	select {
	case <-w.done:
		return true
	default:
		return false
	}
}

func (w *watch) close() {
	w.closeOnce.Do(func() {
		close(w.done)
		if w.stopFromCtx != nil {
			w.stopFromCtx()
		}
		if w.notifier != nil {
			_ = w.notifier.Close()
		}
		w.svc.unregister(w.id)
	})
}

func needsFocusRefresh(path string) bool {
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		return false
	}
	for _, prefix := range []string{"gvfs", "doc"} {
		mount := filepath.Join(runtime, prefix)
		if path == mount || isUnder(path, mount) {
			return true
		}
	}
	return false
}

func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) && rel != "." && !hasParentEscape(rel)
}

func hasParentEscape(rel string) bool {
	return rel == ".." || len(rel) > 3 && rel[:3] == "../"
}
