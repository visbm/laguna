package fs

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type IndexEntry struct {
	FileName string
	Offset   int64
}

// IndexManagerMutex mutex based manager
type IndexManagerMutex struct {
	mu    sync.RWMutex
	index map[uint64]IndexEntry
}

func NewIndexManagerMutex() *IndexManagerMutex {
	return &IndexManagerMutex{
		index: make(map[uint64]IndexEntry),
	}
}

func (im *IndexManagerMutex) Add(lsn uint64, fileName string, offset int64) {
	im.mu.Lock()
	im.index[lsn] = IndexEntry{FileName: fileName, Offset: offset}
	im.mu.Unlock()
}

func (im *IndexManagerMutex) Get(lsn uint64) (IndexEntry, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()
	entry, ok := im.index[lsn]
	if !ok {
		return IndexEntry{}, fmt.Errorf("lsn %d not found", lsn)
	}
	return entry, nil
}

func (im *IndexManagerMutex) AddBatch(entries map[uint64]IndexEntry) {
	im.mu.Lock()
	defer im.mu.Unlock()
	for lsn, entry := range entries {
		im.index[lsn] = entry
	}
}

// IndexManagerCOW COW based manager
type IndexManagerCOW struct {
	current atomic.Value // map[uint64]IndexEntry
}

func NewIndexManagerCOW() *IndexManagerCOW {
	im := &IndexManagerCOW{}
	im.current.Store(make(map[uint64]IndexEntry))
	return im
}

func (im *IndexManagerCOW) Add(lsn uint64, fileName string, offset int64) {
	oldMap := im.current.Load().(map[uint64]IndexEntry)
	newMap := make(map[uint64]IndexEntry, len(oldMap)+1)
	for k, v := range oldMap {
		newMap[k] = v
	}
	newMap[lsn] = IndexEntry{FileName: fileName, Offset: offset}
	im.current.Store(newMap)
}

func (im *IndexManagerCOW) AddBatch(entries map[uint64]IndexEntry) {
	oldMap := im.current.Load().(map[uint64]IndexEntry)

	newMap := make(map[uint64]IndexEntry, len(oldMap)+len(entries))
	for k, v := range oldMap {
		newMap[k] = v
	}
	for lsn, entry := range entries {
		newMap[lsn] = entry
	}

	im.current.Store(newMap)
}

func (im *IndexManagerCOW) Get(lsn uint64) (IndexEntry, error) {
	m := im.current.Load().(map[uint64]IndexEntry)
	entry, ok := m[lsn]
	if !ok {
		return IndexEntry{}, fmt.Errorf("lsn %d not found", lsn)
	}
	return entry, nil
}
