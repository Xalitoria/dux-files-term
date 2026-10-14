package netmount

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"
)

var askTimeout = 2 * time.Minute

const (
	needPassword  = 1 << 0
	needUsername  = 1 << 1
	needDomain    = 1 << 2
	savingAllowed = 1 << 3
	anonAllowed   = 1 << 4
)

type Ask struct {
	Kind          string   `json:"kind"`
	ID            string   `json:"id"`
	Type          string   `json:"type"`
	Message       string   `json:"message"`
	URI           string   `json:"uri"`
	DefaultUser   string   `json:"defaultUser,omitempty"`
	DefaultDomain string   `json:"defaultDomain,omitempty"`
	NeedPassword  bool     `json:"needPassword,omitempty"`
	NeedUsername  bool     `json:"needUsername,omitempty"`
	NeedDomain    bool     `json:"needDomain,omitempty"`
	CanSave       bool     `json:"canSave,omitempty"`
	CanAnonymous  bool     `json:"canAnonymous,omitempty"`
	Choices       []string `json:"choices,omitempty"`
}

type Answer struct {
	Abort     bool
	Password  string
	User      string
	Domain    string
	Anonymous bool
	Save      uint32
	Choice    uint32
}

// mountOp is the object gvfs calls back on while a mount is in flight. One is
// exported per connect so a callback always maps to the request that caused it.
type mountOp struct {
	service  *Service
	uri      string
	password string
	path     dbus.ObjectPath
	counter  atomic.Uint32

	mu      sync.Mutex
	pending map[string]chan Answer
	done    bool
}

func (s *Service) newOp(uri string) *mountOp {
	id := s.opCounter.Add(1)
	return &mountOp{
		service: s,
		uri:     uri,
		path:    dbus.ObjectPath(OpRoot + strconv.FormatUint(uint64(id), 10)),
		pending: map[string]chan Answer{},
	}
}

func (o *mountOp) export(conn *dbus.Conn) error {
	return conn.Export(o, o.path, OpIface)
}

func (o *mountOp) release(conn *dbus.Conn) {
	o.mu.Lock()
	o.done = true
	waiting := o.pending
	o.pending = map[string]chan Answer{}
	o.mu.Unlock()

	for _, reply := range waiting {
		reply <- Answer{Abort: true}
	}
	if conn == nil {
		return
	}
	_ = conn.Export(nil, o.path, OpIface)
}

func (o *mountOp) ask(ask Ask) Answer {
	reply := make(chan Answer, 1)

	o.mu.Lock()
	if o.done {
		o.mu.Unlock()
		return Answer{Abort: true}
	}
	o.pending[ask.ID] = reply
	o.mu.Unlock()

	o.service.publish(ask)

	select {
	case answer := <-reply:
		return answer
	case <-time.After(askTimeout):
		o.forget(ask.ID)
		return Answer{Abort: true}
	}
}

func (o *mountOp) answer(id string, answer Answer) bool {
	o.mu.Lock()
	reply, ok := o.pending[id]
	delete(o.pending, id)
	o.mu.Unlock()

	if !ok {
		return false
	}
	reply <- answer
	return true
}

func (o *mountOp) forget(id string) {
	o.mu.Lock()
	delete(o.pending, id)
	o.mu.Unlock()
}

func (o *mountOp) takePassword(defaultUser string, flags uint32) (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	switch {
	case o.password == "", flags&needPassword == 0, flags&needUsername != 0 && defaultUser == "":
		return "", false
	}
	password := o.password
	o.password = ""
	return password, true
}

func (o *mountOp) nextID() string {
	return string(o.path) + "/" + strconv.FormatUint(uint64(o.counter.Add(1)), 10)
}

func (o *mountOp) AskPassword(message, defaultUser, defaultDomain string, flags uint32) (bool, bool, string, string, string, bool, uint32, *dbus.Error) {
	if password, ok := o.takePassword(defaultUser, flags); ok {
		return true, false, password, defaultUser, defaultDomain, false, 0, nil
	}
	answer := o.ask(Ask{
		Kind:          "ask",
		ID:            o.nextID(),
		Type:          "password",
		Message:       message,
		URI:           o.uri,
		DefaultUser:   defaultUser,
		DefaultDomain: defaultDomain,
		NeedPassword:  flags&needPassword != 0,
		NeedUsername:  flags&needUsername != 0,
		NeedDomain:    flags&needDomain != 0,
		CanSave:       flags&savingAllowed != 0,
		CanAnonymous:  flags&anonAllowed != 0,
	})
	if answer.Abort {
		return true, true, "", "", "", false, 0, nil
	}
	return true, false, answer.Password, answer.User, answer.Domain, answer.Anonymous, answer.Save, nil
}

func (o *mountOp) AskQuestion(message string, choices []string) (bool, bool, uint32, *dbus.Error) {
	answer := o.ask(Ask{
		Kind:    "ask",
		ID:      o.nextID(),
		Type:    "question",
		Message: message,
		URI:     o.uri,
		Choices: choices,
	})
	if answer.Abort {
		return true, true, 0, nil
	}
	return true, false, answer.Choice, nil
}

func (o *mountOp) ShowProcesses(message string, choices []string, processes []int32) (bool, bool, uint32, *dbus.Error) {
	return o.AskQuestion(message, choices)
}

func (o *mountOp) ShowUnmountProgress(message string, timeLeft, bytesLeft int64) *dbus.Error {
	return nil
}

func (o *mountOp) Aborted() *dbus.Error {
	o.release(o.service.connection())
	return nil
}
