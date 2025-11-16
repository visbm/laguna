package inmemory

import (
	"sync"
)

type InMemory struct {
	m     *sync.RWMutex
	store map[string]string
}

func NewInMemory() *InMemory {
	return &InMemory{
		m:     &sync.RWMutex{},
		store: make(map[string]string),
	}
}

func (m *InMemory) Set(key, value string) error {
	m.m.Lock()
	m.store[key] = value
	m.m.Unlock()
	return nil
}

func (m *InMemory) Get(key string) (string, bool) {
	m.m.RLock()
	value, ok := m.store[key]
	m.m.RUnlock()
	return value, ok
}

func (m *InMemory) Del(key string) error {
	m.m.Lock()
	delete(m.store, key)
	m.m.Unlock()
	return nil
}
