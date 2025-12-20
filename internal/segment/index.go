package segment

import (
	"errors"
	"sort"
	"sync"
	"sync/atomic"
)

var ErrNotFound = errors.New("index not found")

type IndexEntry struct {
	LSN      uint64
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

func (im *IndexManagerMutex) GetIndex(lsn uint64) (IndexEntry, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()
	entry, ok := im.index[lsn]
	if !ok {
		return IndexEntry{}, ErrNotFound
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

func (im *IndexManagerCOW) GetIndex(lsn uint64) (IndexEntry, error) {
	m := im.current.Load().(map[uint64]IndexEntry)
	entry, ok := m[lsn]
	if !ok {
		return IndexEntry{}, ErrNotFound
	}
	return entry, nil
}

// IndexManagerArray based on sorted arr
type IndexManagerArray struct {
	mu      sync.RWMutex
	entries []IndexEntry
}

func NewIndexManagerArray() *IndexManagerArray {
	return &IndexManagerArray{
		entries: make([]IndexEntry, 0, 1024),
	}
}

func (im *IndexManagerArray) Add(lsn uint64, fileName string, offset int64) {
	im.mu.Lock()
	defer im.mu.Unlock()

	n := len(im.entries)
	if n == 0 || lsn > im.entries[n-1].LSN {
		im.entries = append(im.entries, IndexEntry{LSN: lsn, FileName: fileName, Offset: offset})
		return
	}

	idx := sort.Search(n, func(i int) bool {
		return im.entries[i].LSN >= lsn
	})

	if idx < n && im.entries[idx].LSN == lsn {
		im.entries[idx].FileName = fileName
		im.entries[idx].Offset = offset
	} else {

		im.entries = append(im.entries, IndexEntry{})
		copy(im.entries[idx+1:], im.entries[idx:])
		im.entries[idx] = IndexEntry{LSN: lsn, FileName: fileName, Offset: offset}
	}
}

func (im *IndexManagerArray) AddBatch(entries []IndexEntry) {

	if len(entries) == 0 {
		return
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	im.entries = append(im.entries, entries...)
}

func (im *IndexManagerArray) GetIndex(lsn uint64) (IndexEntry, error) {
	im.mu.RLock()
	idx := binarySearchIndex(im.entries, lsn)
	im.mu.RUnlock()

	if idx < len(im.entries) && im.entries[idx].LSN == lsn {
		return im.entries[idx], nil
	}
	return IndexEntry{}, ErrNotFound
}

func binarySearchIndex(entries []IndexEntry, lsn uint64) int {
	lo, hi := 0, len(entries)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if entries[mid].LSN < lsn {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}
