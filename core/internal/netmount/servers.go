package netmount

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const recentServers = 12

type Server struct {
	URI      string `json:"uri"`
	Name     string `json:"name,omitempty"`
	Scheme   string `json:"scheme,omitempty"`
	LastUsed int64  `json:"lastUsedMs"`
}

// serverStore keeps the Connect to Server history. Credentials never land
// here; gvfs owns those through the keyring.
type serverStore struct {
	path string
	now  func() time.Time

	mu     sync.Mutex
	loaded bool
	items  []Server
}

func newServerStore(configDir string) *serverStore {
	return &serverStore{path: filepath.Join(configDir, "servers.json"), now: time.Now}
}

func (s *serverStore) list() []Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	return append([]Server(nil), s.items...)
}

func (s *serverStore) remember(server Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()

	server.LastUsed = s.now().UnixMilli()
	kept := []Server{server}
	for _, existing := range s.items {
		if existing.URI == server.URI || len(kept) >= recentServers {
			continue
		}
		kept = append(kept, existing)
	}
	s.items = kept
	s.save()
}

func (s *serverStore) forget(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()

	kept := make([]Server, 0, len(s.items))
	for _, existing := range s.items {
		if existing.URI == uri {
			continue
		}
		kept = append(kept, existing)
	}
	s.items = kept
	s.save()
}

func (s *serverStore) load() {
	if s.loaded {
		return
	}
	s.loaded = true

	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, &s.items)
}

func (s *serverStore) save() {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return
	}
	raw, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return
	}

	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o600); err != nil {
		return
	}
	_ = os.Rename(temp, s.path)
}
