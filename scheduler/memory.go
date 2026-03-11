package scheduler

import (
	"sync"
	"time"
)

type MemoryScheduler struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
}

func NewMemoryScheduler() *MemoryScheduler {
	return &MemoryScheduler{timers: map[string]*time.Timer{}}
}

func (s *MemoryScheduler) Schedule(key string, after time.Duration, fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.timers[key]; ok {
		t.Stop()
	}
	s.timers[key] = time.AfterFunc(after, func() {
		fn()
		s.mu.Lock()
		delete(s.timers, key)
		s.mu.Unlock()
	})
}

func (s *MemoryScheduler) Cancel(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.timers[key]; ok {
		t.Stop()
		delete(s.timers, key)
	}
}
