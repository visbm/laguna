package inmemory

import (
	"hash/fnv"
	"sync"
)

type shard struct {
	m     *sync.RWMutex
	store map[string]string
}

type InMemory struct {
	shards []shard
}

func NewInMemory(s int64) *InMemory {
	sharded := make([]shard, s)
	for i := range sharded {
		sharded[i] = shard{
			m:     &sync.RWMutex{},
			store: make(map[string]string),
		}
	}
	return &InMemory{
		shards: sharded,
	}
}

func (m *InMemory) Set(key, value string) error {
	s, err := m.getShard(key)
	if err != nil {
		return err
	}

	s.m.Lock()
	s.store[key] = value
	s.m.Unlock()
	return nil
}

func (m *InMemory) Get(key string) (string, bool) {
	s, err := m.getShard(key)
	if err != nil {
		return "", false
	}

	s.m.RLock()
	value, ok := s.store[key]
	s.m.RUnlock()
	return value, ok
}

func (m *InMemory) Del(key string) error {
	s, err := m.getShard(key)
	if err != nil {
		return err
	}

	s.m.Lock()
	delete(s.store, key)
	s.m.Unlock()
	return nil
}

func (m *InMemory) getShard(key string) (*shard, error) {
	hash := fnv.New64a()
	_, err := hash.Write([]byte(key))
	if err != nil {
		return nil, err
	}

	shardID := hash.Sum64() % uint64(len(m.shards))

	return &m.shards[shardID], nil
}
