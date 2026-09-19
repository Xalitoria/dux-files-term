package files

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBus struct {
	mu     sync.Mutex
	events []batch
}

func (b *fakeBus) Publish(_ string, data any) {
	payload, ok := data.(batch)
	if !ok {
		return
	}
	b.mu.Lock()
	b.events = append(b.events, payload)
	b.mu.Unlock()
}

func (b *fakeBus) snapshot() []batch {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]batch, len(b.events))
	copy(out, b.events)
	return out
}

func (b *fakeBus) await(t *testing.T, match func(batch) bool) batch {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, event := range b.snapshot() {
			if match(event) {
				return event
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no matching event, got %+v", b.snapshot())
	return batch{}
}

type fakeNotifier struct {
	events chan fsnotify.Event
	errors chan error
	closed chan struct{}
	once   sync.Once
}

func newFakeNotifier() *fakeNotifier {
	return &fakeNotifier{events: make(chan fsnotify.Event, 64), errors: make(chan error, 8), closed: make(chan struct{})}
}

func (f *fakeNotifier) Add(string) error              { return nil }
func (f *fakeNotifier) Events() <-chan fsnotify.Event { return f.events }
func (f *fakeNotifier) Errors() <-chan error          { return f.errors }
func (f *fakeNotifier) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc := NewService(&fakeBus{}, t.TempDir())
	t.Cleanup(svc.Close)
	return svc
}

func newWatchedDir(t *testing.T) (*Service, *fakeBus, string, *watch) {
	t.Helper()
	bus := &fakeBus{}
	svc := NewService(bus, t.TempDir())
	t.Cleanup(svc.Close)

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "seed.txt"), []byte("seed"), 0o644))

	w, err := svc.Watch(context.Background(), root, nameSort(), false)
	require.NoError(t, err)
	require.True(t, w.watching)
	svc.Attach([]string{w.topic})
	return svc, bus, root, w
}

func TestWatchReportsCreateWriteDelete(t *testing.T) {
	_, bus, root, w := newWatchedDir(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, "new.txt"), []byte("hello"), 0o644))
	added := bus.await(t, func(b batch) bool { return len(b.Added) > 0 && b.Added[0].Name == "new.txt" })
	assert.Equal(t, "batch", added.Kind)
	assert.Equal(t, w.id, added.WatchID)
	assert.Positive(t, added.Seq)
	assert.Equal(t, int64(5), added.Added[0].Size)

	require.NoError(t, os.WriteFile(filepath.Join(root, "new.txt"), []byte("hello again"), 0o644))
	changed := bus.await(t, func(b batch) bool {
		return len(b.Changed) > 0 && b.Changed[0].Name == "new.txt" && b.Changed[0].Size == 11
	})
	assert.Greater(t, changed.Seq, added.Seq)

	require.NoError(t, os.Remove(filepath.Join(root, "new.txt")))
	removed := bus.await(t, func(b batch) bool { return len(b.Removed) > 0 && b.Removed[0] == "new.txt" })
	assert.Greater(t, removed.Seq, changed.Seq)

	page := w.page(ListOptions{Sort: nameSort()})
	assert.Equal(t, []string{"seed.txt"}, names(page.Entries))
}

func TestWatchRenameIsRemoveAndAdd(t *testing.T) {
	_, bus, root, w := newWatchedDir(t)

	require.NoError(t, os.Rename(filepath.Join(root, "seed.txt"), filepath.Join(root, "renamed.txt")))
	bus.await(t, func(b batch) bool { return len(b.Removed) > 0 && b.Removed[0] == "seed.txt" })
	bus.await(t, func(b batch) bool { return len(b.Added) > 0 && b.Added[0].Name == "renamed.txt" })

	page := w.page(ListOptions{Sort: nameSort()})
	assert.Equal(t, []string{"renamed.txt"}, names(page.Entries))
}

func TestWatchBatchCarriesInsertIndex(t *testing.T) {
	bus := &fakeBus{}
	svc := NewService(bus, t.TempDir())
	t.Cleanup(svc.Close)

	root := t.TempDir()
	for _, name := range []string{"a.txt", "c.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), nil, 0o644))
	}
	w, err := svc.Watch(context.Background(), root, nameSort(), false)
	require.NoError(t, err)
	svc.Attach([]string{w.topic})

	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), nil, 0o644))
	added := bus.await(t, func(b batch) bool { return len(b.Added) > 0 && b.Added[0].Name == "b.txt" })
	assert.Equal(t, []int{1}, added.AddedAt)

	require.NoError(t, os.Mkdir(filepath.Join(root, "sub"), 0o755))
	dir := bus.await(t, func(b batch) bool { return len(b.Added) > 0 && b.Added[0].Name == "sub" })
	assert.Equal(t, []int{0}, dir.AddedAt, "a directory lands above the files")
	assert.Equal(t, []string{"sub", "a.txt", "b.txt", "c.txt"}, names(w.page(ListOptions{Sort: nameSort()}).Entries))
}

func TestWatchReportsAMoveAsRemoveAndAdd(t *testing.T) {
	bus := &fakeBus{}
	svc := NewService(bus, t.TempDir())
	t.Cleanup(svc.Close)

	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), nil, 0o644))
	}
	w, err := svc.Watch(context.Background(), root, SortSpec{Key: SortSize, DirsFirst: true}, false)
	require.NoError(t, err)
	svc.Attach([]string{w.topic})

	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("now the biggest"), 0o644))
	moved := bus.await(t, func(b batch) bool { return len(b.Added) > 0 && b.Added[0].Name == "a.txt" })
	assert.Equal(t, []string{"a.txt"}, moved.Removed)
	assert.Equal(t, []int{1}, moved.AddedAt)
	assert.Equal(t, []string{"b.txt", "a.txt"}, names(w.page(ListOptions{Sort: SortSpec{Key: SortSize, DirsFirst: true}}).Entries))
}

func TestWatchCoalescesBurstsAndKeepsSeqOrdered(t *testing.T) {
	_, bus, root, _ := newWatchedDir(t)

	for i := range 20 {
		require.NoError(t, os.WriteFile(filepath.Join(root, "burst"+string(rune('a'+i))+".txt"), nil, 0o644))
	}
	bus.await(t, func(b batch) bool { return len(b.Added) >= 10 })

	time.Sleep(2 * coalesceWindow)
	events := bus.snapshot()
	assert.Less(t, len(events), 20, "a burst must not become one event per file")
	for i := 1; i < len(events); i++ {
		assert.Equal(t, events[i-1].Seq+1, events[i].Seq, "sequence numbers must have no gaps")
	}
}

func TestWatchHidesHiddenEntriesUntilRequested(t *testing.T) {
	_, bus, root, w := newWatchedDir(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, ".dotfile"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "visible.txt"), nil, 0o644))
	bus.await(t, func(b batch) bool { return len(b.Added) > 0 && b.Added[0].Name == "visible.txt" })

	for _, event := range bus.snapshot() {
		for _, entry := range event.Added {
			assert.NotEqual(t, ".dotfile", entry.Name)
		}
	}
	assert.Contains(t, names(w.page(ListOptions{Sort: nameSort(), IncludeHidden: true}).Entries), ".dotfile")
}

func TestWatchResyncOnOverflow(t *testing.T) {
	bus := &fakeBus{}
	svc := NewService(bus, t.TempDir())
	t.Cleanup(svc.Close)

	fake := newFakeNotifier()
	svc.newNotifier = func() (notifier, error) { return fake, nil }

	root := t.TempDir()
	w, err := svc.Watch(context.Background(), root, nameSort(), false)
	require.NoError(t, err)
	svc.Attach([]string{w.topic})

	require.NoError(t, os.WriteFile(filepath.Join(root, "late.txt"), nil, 0o644))
	fake.errors <- fsnotify.ErrEventOverflow

	resync := bus.await(t, func(b batch) bool { return b.Kind == "resync" })
	assert.Equal(t, w.id, resync.WatchID)
	assert.Equal(t, []string{"late.txt"}, names(w.page(ListOptions{Sort: nameSort()}).Entries))
}

func TestWatchGoneWhenDirectoryDisappears(t *testing.T) {
	bus := &fakeBus{}
	svc := NewService(bus, t.TempDir())
	t.Cleanup(svc.Close)

	root := filepath.Join(t.TempDir(), "doomed")
	require.NoError(t, os.Mkdir(root, 0o755))
	w, err := svc.Watch(context.Background(), root, nameSort(), false)
	require.NoError(t, err)
	svc.Attach([]string{w.topic})

	require.NoError(t, os.Remove(root))
	gone := bus.await(t, func(b batch) bool { return b.Kind == "gone" })
	assert.Equal(t, w.id, gone.WatchID)

	_, live := svc.watchByID(w.id)
	assert.False(t, live, "a gone watch unregisters itself")
}

func TestWatchClosesWithItsConnection(t *testing.T) {
	svc := newTestService(t)
	ctx, cancel := context.WithCancel(context.Background())

	w, err := svc.Watch(ctx, t.TempDir(), nameSort(), false)
	require.NoError(t, err)

	cancel()
	require.Eventually(t, func() bool {
		_, live := svc.watchByID(w.id)
		return !live
	}, time.Second, 10*time.Millisecond)
	assert.True(t, w.closed())
}

func TestWatchBuffersEventsUntilSubscribed(t *testing.T) {
	bus := &fakeBus{}
	svc := NewService(bus, t.TempDir())
	t.Cleanup(svc.Close)

	root := t.TempDir()
	w, err := svc.Watch(context.Background(), root, nameSort(), false)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, "early.txt"), nil, 0o644))
	require.Eventually(t, func() bool { return w.sequence() > 0 }, time.Second, 10*time.Millisecond)
	assert.Empty(t, bus.snapshot(), "nothing is published before a subscriber exists")

	svc.Attach([]string{w.topic})
	added := bus.await(t, func(b batch) bool { return len(b.Added) > 0 && b.Added[0].Name == "early.txt" })
	assert.Equal(t, 1, added.Seq)
}

func TestWatchDropsOverflowingBufferForOneResync(t *testing.T) {
	bus := &fakeBus{}
	svc := NewService(bus, t.TempDir())
	t.Cleanup(svc.Close)

	root := t.TempDir()
	w, err := svc.Watch(context.Background(), root, nameSort(), false)
	require.NoError(t, err)

	w.mu.Lock()
	for range pendingCap {
		w.pending = append(w.pending, batch{Kind: "batch"})
	}
	w.publishLocked(batch{Kind: "batch", Removed: []string{"anything"}})
	w.mu.Unlock()

	svc.Attach([]string{w.topic})
	events := bus.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "resync", events[0].Kind)
}

func TestWatchLeavesNoGoroutinesBehind(t *testing.T) {
	svc := newTestService(t)
	root := t.TempDir()
	for i := range 50 {
		require.NoError(t, os.WriteFile(filepath.Join(root, "f"+string(rune('a'+i%26))+string(rune('a'+i/26))+".txt"), []byte("x"), 0o644))
	}

	settle(t)
	baseline := runtime.NumGoroutine()

	for range 5 {
		w, err := svc.Watch(context.Background(), root, nameSort(), false)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(root, "touch.txt"), []byte("y"), 0o644))
		w.close()
	}
	svc.pool.drain()
	settle(t)

	assert.LessOrEqual(t, runtime.NumGoroutine(), baseline, "watches and the worker pool must idle at zero goroutines")
}

func settle(t *testing.T) {
	t.Helper()
	for range 50 {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
}
