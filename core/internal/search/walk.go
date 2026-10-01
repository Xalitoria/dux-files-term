package search

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AvengeMedia/dankgo/files"
)

const (
	batchSize     = 200
	flushInterval = 100 * time.Millisecond
)

type event struct {
	SearchID   string        `json:"searchId"`
	Kind       string        `json:"kind"`
	Entries    []files.Entry `json:"entries,omitempty"`
	Count      int           `json:"count"`
	Truncated  bool          `json:"truncated,omitempty"`
	Cancelled  bool          `json:"cancelled,omitempty"`
	Unreadable int           `json:"unreadable,omitempty"`
}

type search struct {
	id, topic string
	svc       *Service
	ctx       context.Context
	cancel    context.CancelFunc
	finished  chan struct{}

	mu       sync.Mutex
	attached bool
	done     bool
	pending  []event
	stopWait func() bool
}

func newSearch(svc *Service) *search {
	ctx, cancel := context.WithCancel(context.Background())
	return &search{svc: svc, ctx: ctx, cancel: cancel, finished: make(chan struct{})}
}

// follow ends the search with the connection that started it; it runs after
// register so the id is settled.
func (sc *search) follow(parent context.Context) {
	sc.stopWait = context.AfterFunc(parent, func() {
		sc.cancel()
		sc.svc.unregister(sc.id)
	})
}

func (sc *search) run(root, query string, opts Options) {
	defer close(sc.finished)
	defer sc.retire()

	w := &walker{
		sc:      sc,
		query:   strings.ToLower(query),
		opts:    opts,
		limit:   opts.limit(),
		matches: make(chan files.Entry, batchSize),
	}
	if !opts.CrossDevice {
		if dev, ok := deviceOf(root); ok {
			w.device, w.sameDevice = dev, true
		}
	}

	collected := make(chan event, 1)
	go w.collect(collected)

	w.queue.push(root)
	var workers sync.WaitGroup
	for range sc.svc.workers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			w.work()
		}()
	}
	go func() {
		w.queue.wait()
		w.queue.close()
	}()
	workers.Wait()
	close(w.matches)

	done := <-collected
	done.Cancelled = sc.ctx.Err() != nil && !done.Truncated
	done.Unreadable = int(w.unreadable.Load())
	sc.publish(done)
}

func (sc *search) publish(ev event) {
	ev.SearchID = sc.id
	sc.mu.Lock()
	if !sc.attached && !sc.svc.attachOnOpen.Load() {
		sc.pending = append(sc.pending, ev)
		sc.mu.Unlock()
		return
	}
	sc.mu.Unlock()
	sc.svc.publish(sc.topic, ev)
}

// attach flushes the backlog under the lock so a batch the collector emits
// meanwhile cannot overtake it.
func (sc *search) attach() {
	sc.mu.Lock()
	sc.attached = true
	for _, ev := range sc.pending {
		sc.svc.publish(sc.topic, ev)
	}
	sc.pending = nil
	done := sc.done
	sc.mu.Unlock()
	if done {
		sc.drop()
	}
}

// retire drops the search from the registry once nothing can still ask for
// its buffered events: a search that finished before its topic had a
// subscriber stays until Attach flushes it, Cancel drops it, or the client
// goes away (the disconnect hook stays armed for that).
func (sc *search) retire() {
	sc.mu.Lock()
	sc.done = true
	flushed := sc.attached || sc.svc.attachOnOpen.Load()
	sc.mu.Unlock()
	if flushed {
		sc.drop()
	}
}

func (sc *search) drop() {
	if sc.stopWait != nil {
		sc.stopWait()
	}
	sc.svc.unregister(sc.id)
}

type walker struct {
	sc         *search
	query      string
	opts       Options
	limit      int
	device     uint64
	sameDevice bool
	queue      dirQueue
	matches    chan files.Entry
	unreadable atomic.Int64
}

func (w *walker) work() {
	for {
		dir, ok := w.queue.pop()
		if !ok {
			return
		}
		w.readDir(dir)
		w.queue.done()
	}
}

func (w *walker) readDir(dir string) {
	if w.sc.ctx.Err() != nil {
		return
	}
	children, err := os.ReadDir(dir)
	if err != nil {
		w.unreadable.Add(1)
		return
	}
	for _, child := range children {
		if w.sc.ctx.Err() != nil {
			return
		}
		name := child.Name()
		if !w.opts.IncludeHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dir, name)
		if child.Type()&fs.ModeSymlink == 0 && child.IsDir() && w.descends(full) {
			w.queue.push(full)
		}
		if w.query != "" && !strings.Contains(strings.ToLower(name), w.query) {
			continue
		}
		w.emit(full)
	}
}

func (w *walker) descends(dir string) bool {
	if !w.sameDevice {
		return true
	}
	dev, ok := deviceOf(dir)
	return !ok || dev == w.device
}

func (w *walker) emit(path string) {
	entry, err := w.sc.svc.stater.Stat(path)
	if err != nil || !w.opts.accepts(entry) {
		return
	}
	select {
	case w.matches <- entry:
	case <-w.sc.ctx.Done():
	}
}

// collect batches matches into events, flushing on size or after a short
// pause, and stops the walk at the cap.
func (w *walker) collect(out chan<- event) {
	var batch []files.Entry
	count := 0
	truncated := false
	var timer *time.Timer
	var due <-chan time.Time

	flush := func() {
		if len(batch) == 0 {
			return
		}
		w.sc.publish(event{Kind: "results", Entries: batch, Count: count})
		batch = nil
	}
	arm := func() {
		if timer == nil {
			timer = time.NewTimer(flushInterval)
			due = timer.C
			return
		}
		timer.Reset(flushInterval)
	}

	for {
		select {
		case entry, ok := <-w.matches:
			if !ok {
				flush()
				if timer != nil {
					timer.Stop()
				}
				out <- event{Kind: "done", Count: count, Truncated: truncated}
				return
			}
			if truncated {
				continue
			}
			batch = append(batch, entry)
			count++
			switch {
			case count >= w.limit:
				truncated = true
				w.sc.cancel()
				flush()
			case len(batch) == 1:
				arm()
			case len(batch) >= batchSize:
				flush()
			}
		case <-due:
			flush()
		}
	}
}
