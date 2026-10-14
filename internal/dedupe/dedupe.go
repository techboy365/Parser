package dedupe

import (
	"sync"

	"github.com/techboy365/Parser/internal/normalize"
)

type Store struct {
	mu   sync.RWMutex
	seen map[string]struct{}
}

func NewStore() *Store {
	return &Store{seen: make(map[string]struct{})}
}

func (s *Store) Add(raw string) (string, bool) {
	normalized, err := normalize.URL(raw)
	if err != nil || normalized == "" {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[normalized]; ok {
		return normalized, false
	}
	s.seen[normalized] = struct{}{}
	return normalized, true
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.seen)
}

func (s *Store) Snapshot() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.seen))
	for u := range s.seen {
		out = append(out, u)
	}
	return out
}
