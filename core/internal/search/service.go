package search

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/AvengeMedia/dankgo/files"
)

const (
	MaxResults    = 10000
	MaxConcurrent = 8
	topicPrefix   = "search:"
)

type Publisher interface {
	Publish(topic string, data any)
}

type Stater interface {
	Stat(path string) (files.Entry, error)
}

type Service struct {
	bus          Publisher
	stater       Stater
	workers      int
	attachOnOpen atomic.Bool

	mu       sync.Mutex
	searches map[string]*search
	nextID   int
}

func NewService(bus Publisher, stater Stater) *Service {
	return &Service{
		bus:      bus,
		stater:   stater,
		workers:  min(4, runtime.NumCPU()),
		searches: map[string]*search{},
	}
}

// Attach flushes what a search buffered before its topic had a subscriber.
func (s *Service) Attach(topics []string) {
	for _, topic := range topics {
		id, ok := strings.CutPrefix(topic, topicPrefix)
		if !ok {
			continue
		}
		if sc, live := s.byID(id); live {
			sc.attach()
		}
	}
}

// AttachOnOpen is for hosts without per-topic subscribe; searches never buffer.
func (s *Service) AttachOnOpen() { s.attachOnOpen.Store(true) }

func (s *Service) Close() {
	s.mu.Lock()
	all := make([]*search, 0, len(s.searches))
	for _, sc := range s.searches {
		all = append(all, sc)
	}
	s.mu.Unlock()
	for _, sc := range all {
		sc.cancel()
		<-sc.finished
	}
}

// Start walks root for entries matching query and opts, streaming batches on
// the returned topic. The search ends with the connection context.
func (s *Service) Start(ctx context.Context, root, query string, opts Options) (id, topic string, err error) {
	if err := validate(root, query, opts); err != nil {
		return "", "", err
	}

	sc := newSearch(s)
	if err := s.register(sc); err != nil {
		return "", "", err
	}
	sc.follow(ctx)
	go sc.run(root, query, opts)
	return sc.id, sc.topic, nil
}

func (s *Service) Cancel(id string) bool {
	sc, ok := s.byID(id)
	if !ok {
		return false
	}
	sc.cancel()
	sc.mu.Lock()
	done := sc.done
	sc.mu.Unlock()
	if done {
		sc.drop()
	}
	return true
}

func (s *Service) register(sc *search) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.searches) >= MaxConcurrent {
		return &files.Error{Code: files.CodeBusy, Err: errors.New("too many searches running")}
	}
	s.nextID++
	sc.id = "s" + strconv.Itoa(s.nextID)
	sc.topic = topicPrefix + sc.id
	s.searches[sc.id] = sc
	return nil
}

func (s *Service) unregister(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.searches, id)
}

func (s *Service) byID(id string) (*search, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.searches[id]
	return sc, ok
}

func (s *Service) publish(topic string, payload any) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(topic, payload)
}
