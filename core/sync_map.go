package core

import "sync"

// SyncMap is an implementation of a map that is safe for concurrent usage.
type SyncMap[K comparable, V any] struct {
	m  map[K]V
	mu sync.RWMutex
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		m: make(map[K]V),
	}
}

func (s *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok = s.m[key]
	return
}

// LoadAndStore retrieves the value for a key, applies the function f to it, and stores the result.
// It guarantees that the whole operation is atomic.
func (s *SyncMap[K, V]) LoadAndStore(key K, f func(value V, ok bool) V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.m[key]
	s.m[key] = f(value, ok)
}

func (s *SyncMap[K, V]) Store(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

func (s *SyncMap[K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

func (s *SyncMap[K, V]) RRange(f func(key K, value V) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.m {
		if !f(k, v) {
			break
		}
	}
}

func (s *SyncMap[K, V]) WRange(f func(key K, value V) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.m {
		if !f(k, v) {
			break
		}
	}
}
