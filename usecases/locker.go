package usecases

import "sync"

type KeyedLocker struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewKeyedLocker() *KeyedLocker {
	return &KeyedLocker{locks: map[string]*sync.Mutex{}}
}

func (l *KeyedLocker) WithKey(key string, fn func() error) error {
	lk := l.get(key)
	lk.Lock()
	defer lk.Unlock()
	return fn()
}

func (l *KeyedLocker) get(key string) *sync.Mutex {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lk, ok := l.locks[key]; ok {
		return lk
	}
	lk := &sync.Mutex{}
	l.locks[key] = lk
	return lk
}
