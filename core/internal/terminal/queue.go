package terminal

import (
	"io"
	"sync"
)

// Decouples the emulator's input pipe from the pty so a shell that stops reading never stalls output parsing.
type writeQueue struct {
	out    io.Writer
	mu     sync.Mutex
	ready  *sync.Cond
	chunks [][]byte
	closed bool
}

func newWriteQueue(out io.Writer) *writeQueue {
	q := &writeQueue{out: out}
	q.ready = sync.NewCond(&q.mu)
	return q
}

func (q *writeQueue) push(chunk []byte) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.chunks = append(q.chunks, chunk)
	q.ready.Signal()
}

func (q *writeQueue) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.chunks = nil
	q.ready.Signal()
}

func (q *writeQueue) run() {
	for {
		q.mu.Lock()
		for len(q.chunks) == 0 && !q.closed {
			q.ready.Wait()
		}
		if q.closed {
			q.mu.Unlock()
			return
		}
		chunk := q.chunks[0]
		q.chunks = q.chunks[1:]
		q.mu.Unlock()
		if _, err := q.out.Write(chunk); err != nil {
			q.close()
			return
		}
	}
}
