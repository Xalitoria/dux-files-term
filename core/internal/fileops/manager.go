package fileops

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
)

const (
	maxConcurrent = 2
	finishedCap   = 20
	undoCap       = 50
	tickInterval  = 500 * time.Millisecond
)

type Publisher interface {
	Publish(topic string, data any)
}

const Topic = "jobs"

type event struct {
	Kind     string    `json:"kind"`
	JobID    string    `json:"jobId,omitempty"`
	Job      *View     `json:"job,omitempty"`
	Progress *Progress `json:"progress,omitempty"`
	Conflict *Conflict `json:"conflict,omitempty"`
	Failure  *Failure  `json:"failure,omitempty"`
}

type undoRecord struct {
	kind     Kind
	pairs    []pair
	guards   []guard
	trashDir string
}

type guard struct {
	size    int64
	mtimeMs int64
}

type Manager struct {
	bus      Publisher
	stater   Stater
	elevator Elevator
	slots    chan struct{}

	mu       sync.Mutex
	nextID   int
	active   map[string]*job
	order    []string
	finished []View
	undo     map[string][]*undoRecord
}

func NewManager(bus Publisher, stater Stater) *Manager {
	return &Manager{
		bus:    bus,
		stater: stater,
		slots:  make(chan struct{}, maxConcurrent),
		active: map[string]*job{},
		undo:   map[string][]*undoRecord{},
	}
}

// SetElevator installs the "retry as administrator" route; call it before
// the first job.
func (m *Manager) SetElevator(e Elevator) {
	m.elevator = e
}

type Elevation struct {
	Available bool   `json:"available"`
	Route     string `json:"route,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func (m *Manager) Elevation() Elevation {
	if m.elevator == nil {
		return Elevation{Reason: "NOROUTE"}
	}
	err := m.elevator.Available()
	if err == nil {
		return Elevation{Available: true, Route: m.elevator.Route()}
	}
	var unavailable *Unavailable
	if errors.As(err, &unavailable) {
		return Elevation{Route: m.elevator.Route(), Reason: unavailable.Reason}
	}
	return Elevation{Route: m.elevator.Route(), Reason: err.Error()}
}

func (m *Manager) elevationAvailable() bool {
	return m.elevator != nil && m.elevator.Available() == nil
}

func (m *Manager) enqueue(s spec) *job {
	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.nextID++
	j := &job{
		id:      "j" + strconv.Itoa(m.nextID),
		spec:    s,
		mgr:     m,
		ctx:     ctx,
		cancel:  cancel,
		answers: make(chan Answer, 1),
		state:   StateQueued,
	}
	m.active[j.id] = j
	m.order = append(m.order, j.id)
	m.mu.Unlock()

	m.publishJob(j.view())
	go m.run(j)
	return j
}

func (m *Manager) run(j *job) {
	defer j.cancel()

	select {
	case m.slots <- struct{}{}:
	case <-j.ctx.Done():
		j.finish(nil)
		return
	}
	defer func() { <-m.slots }()

	j.setState(StateRunning)
	j.finish(execute(j))
}

func (j *job) finish(err error) {
	j.mu.Lock()
	j.cancelled = j.cancelled || j.ctx.Err() != nil
	j.terminal = true
	j.endedMs = time.Now().UnixMilli()
	j.conflict = nil
	j.failure = nil
	j.pendingID = ""
	switch {
	case err != nil:
		j.state = StateFailed
		j.errMsg = err.Error()
		j.errCode = codeOf(err)
	default:
		j.state = StateDone
	}
	record := j.record
	scope := j.spec.scope
	view := j.viewLocked()
	j.mu.Unlock()

	if record != nil && len(record.pairs) > 0 && !j.spec.undo {
		j.mgr.pushUndo(scope, record)
	}
	j.mgr.retire(view)
	j.mgr.publishJob(view)
}

func (m *Manager) retire(view View) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, view.ID)
	for i, id := range m.order {
		if id != view.ID {
			continue
		}
		m.order = append(m.order[:i], m.order[i+1:]...)
		break
	}
	m.finished = append(m.finished, view)
	if len(m.finished) > finishedCap {
		m.finished = m.finished[len(m.finished)-finishedCap:]
	}
}

func (m *Manager) pushUndo(scope string, record *undoRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stack := append(m.undo[scope], record)
	if len(stack) > undoCap {
		stack = stack[len(stack)-undoCap:]
	}
	m.undo[scope] = stack
}

func (m *Manager) popUndo(scope string) *undoRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	stack := m.undo[scope]
	if len(stack) == 0 {
		delete(m.undo, scope)
		return nil
	}
	record := stack[len(stack)-1]
	if len(stack) == 1 {
		delete(m.undo, scope)
		return record
	}
	m.undo[scope] = stack[:len(stack)-1]
	return record
}

func (m *Manager) Undo(scope string) (View, error) {
	record := m.popUndo(scope)
	if record == nil {
		return View{}, errors.New("nothing to undo")
	}

	s := spec{
		kind:     record.kind,
		scope:    scope,
		pairs:    record.pairs,
		guards:   record.guards,
		trashDir: record.trashDir,
		undo:     true,
	}
	for _, p := range record.pairs {
		s.sources = append(s.sources, p.src)
	}
	return m.enqueue(s).view(), nil
}

func (m *Manager) Jobs() []View {
	m.mu.Lock()
	defer m.mu.Unlock()

	views := make([]View, 0, len(m.order)+len(m.finished))
	for _, id := range m.order {
		views = append(views, m.active[id].view())
	}
	return append(views, m.finished...)
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	j, ok := m.active[id]
	m.mu.Unlock()
	if !ok {
		return errors.New("unknown job: " + id)
	}

	j.mu.Lock()
	if j.terminal {
		j.mu.Unlock()
		return errors.New("job already finished: " + id)
	}
	j.cancelled = true
	j.mu.Unlock()

	j.setState(StateCancelling)
	j.cancel()
	return nil
}

func (m *Manager) Respond(jobID string, answer Answer) error {
	m.mu.Lock()
	j, ok := m.active[jobID]
	m.mu.Unlock()
	if !ok {
		return errors.New("unknown job: " + jobID)
	}

	j.mu.Lock()
	claimed := j.pendingID != "" && j.pendingID == answer.ID
	if claimed {
		j.pendingID = ""
	}
	j.mu.Unlock()
	if !claimed {
		return errors.New("job is not waiting for that answer: " + jobID)
	}

	select {
	case j.answers <- answer:
		return nil
	case <-j.ctx.Done():
		return errors.New("job is no longer running: " + jobID)
	}
}

func (m *Manager) publish(payload event) {
	if m.bus == nil {
		return
	}
	m.bus.Publish(Topic, payload)
}

func (m *Manager) publishJob(view View) {
	m.publish(event{Kind: "job", JobID: view.ID, Job: &view})
}

// ask pauses the job until the UI answers, or until it is cancelled.
func (j *job) ask(payload event, state State) Answer {
	j.mu.Lock()
	j.conflict = payload.Conflict
	j.failure = payload.Failure
	j.pendingID = payload.answerID()
	j.state = state
	view := j.viewLocked()
	j.mu.Unlock()

	j.mgr.publish(payload)
	j.mgr.publishJob(view)

	for {
		select {
		case answer := <-j.answers:
			if answer.ID != payload.answerID() {
				continue
			}
			j.mu.Lock()
			j.conflict = nil
			j.failure = nil
			j.pendingID = ""
			j.mu.Unlock()
			j.setState(StateRunning)
			if answer.Action == ActionCancel {
				j.cancel()
			}
			return answer
		case <-j.ctx.Done():
			j.mu.Lock()
			j.pendingID = ""
			j.mu.Unlock()
			return Answer{Action: ActionCancel}
		}
	}
}

func (e event) answerID() string {
	switch {
	case e.Conflict != nil:
		return e.Conflict.ID
	case e.Failure != nil:
		return e.Failure.ID
	}
	return ""
}

func (j *job) tick(force bool) {
	j.mu.Lock()
	now := time.Now()
	if !force && now.Sub(j.lastTick) < tickInterval {
		j.mu.Unlock()
		return
	}
	if elapsed := now.Sub(j.lastTick); elapsed > 0 && !j.lastTick.IsZero() {
		j.progress.BytesPerSec = int64(float64(j.progress.BytesDone-j.lastBytes) / elapsed.Seconds())
	}
	j.lastTick = now
	j.lastBytes = j.progress.BytesDone
	progress := j.progress
	j.mu.Unlock()

	j.mgr.publish(event{Kind: "progress", JobID: j.id, Progress: &progress})
}
