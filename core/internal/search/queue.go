package search

import "sync"

// dirQueue hands directories to a bounded set of workers without ever
// blocking the worker that discovered them.
type dirQueue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	items   []string
	closed  bool
	pending sync.WaitGroup
}

func (q *dirQueue) init() {
	if q.cond == nil {
		q.cond = sync.NewCond(&q.mu)
	}
}

func (q *dirQueue) push(dir string) {
	q.mu.Lock()
	q.init()
	q.pending.Add(1)
	q.items = append(q.items, dir)
	q.cond.Signal()
	q.mu.Unlock()
}

func (q *dirQueue) pop() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.init()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return "", false
	}
	dir := q.items[0]
	q.items = q.items[1:]
	return dir, true
}

func (q *dirQueue) done() { q.pending.Done() }

func (q *dirQueue) wait() { q.pending.Wait() }

func (q *dirQueue) close() {
	q.mu.Lock()
	q.init()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}
