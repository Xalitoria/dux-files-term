package search

import "sync"

type dirItem struct {
	path     string
	relLower string
	depth    int
	root     *rootInfo
	patterns []ignorePattern
}

type dirQueue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	items   []dirItem
	closed  bool
	pending sync.WaitGroup
}

func (q *dirQueue) init() {
	if q.cond == nil {
		q.cond = sync.NewCond(&q.mu)
	}
}

func (q *dirQueue) push(item dirItem) {
	q.mu.Lock()
	q.init()
	q.pending.Add(1)
	q.items = append(q.items, item)
	q.cond.Signal()
	q.mu.Unlock()
}

func (q *dirQueue) pop() (dirItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.init()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return dirItem{}, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
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
