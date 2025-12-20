package segment

import (
	"laguna/common/logger"
	"laguna/internal/database/wal"
	"laguna/utils/fs"
	"sync"
)

type IndexManager interface {
	AddBatch(entries map[uint64]IndexEntry)
	GetIndex(lsn uint64) (IndexEntry, error)
}

type Manager struct {
	m      *sync.RWMutex
	curSeg *FileSegment

	im  IndexManager
	log logger.Logger
}

func NewSegmentManager(log logger.Logger, im IndexManager, segment *FileSegment) *Manager {
	return &Manager{
		m:      &sync.RWMutex{},
		curSeg: segment,
		log:    log,
		im:     im,
	}
}

func (s *Manager) Write(rows []*wal.Row) error {
	batch, err := s.createButch(rows)
	if err != nil {
		return err
	}

	return s.processBatches(batch, rows)
}
func (s *Manager) GetIndex(lsn uint64) (IndexEntry, error) {
	return s.im.GetIndex(lsn)
}

func (s *Manager) processBatches(batch [][]byte, rows []*wal.Row) error {
	batchSize := 0
	for _, b := range batch {
		batchSize += len(b) + 1
	}

	if s.curSeg.Fits(int64(batchSize)) {
		err := s.writeBatch(batch, rows, batchSize)
		if err != nil {
			s.log.Error("error writing to file", logger.Error(err))
			return err
		}

		return nil
	}

	currentSize := 0
	start := 0

	for i, b := range batch {
		size := len(b)

		if !s.curSeg.Fits(int64(currentSize + size)) {
			err := s.writeBatch(batch[start:i], rows[start:i], currentSize)
			if err != nil {
				s.log.Error("error writing to file", logger.Error(err))
				return err
			}

			s.m.Lock()
			err = s.curSeg.Rotate()
			if err != nil {
				s.log.Error("error writing to file", logger.Error(err))
				return err
			}
			s.m.Unlock()

			start = i
			currentSize = 0
		}

		currentSize += size
	}

	err := s.writeBatch(batch[start:], rows[start:], currentSize)
	if err != nil {
		s.log.Error("error writing to file", logger.Error(err))
		return err
	}

	return nil
}

func (s *Manager) writeBatch(batch [][]byte, rows []*wal.Row, bufSize int) error {
	name := s.curSeg.GetName()
	offset := s.curSeg.CurrentOffset()

	entries := make(map[uint64]IndexEntry, len(rows))
	for i, row := range rows {
		entries[row.GetLsnID()] = IndexEntry{FileName: name, Offset: offset}
		offset += int64(len(batch[i]))
	}

	data := s.flatten(batch, bufSize)
	err := s.writeInSeg(data)
	if err != nil {
		s.log.Error("error writing to file", logger.Error(err))
		return err
	}

	go s.AddBatchIndex(entries)

	return nil
}
func (s *Manager) AddBatchIndex(entries map[uint64]IndexEntry) {
	s.im.AddBatch(entries)
}

func (s *Manager) flatten(batch [][]byte, bufSize int) []byte {
	buf := make([]byte, 0, bufSize)
	for _, b := range batch {
		buf = append(buf, b...)
	}

	return buf
}

func (s *Manager) writeInSeg(batch []byte) error {
	if len(batch) == 0 {
		return nil
	}

	s.m.Lock()
	err := s.curSeg.Write(batch)
	if err != nil {
		s.log.Error("error writing to file", logger.Error(err))
		return err
	}
	s.m.Unlock()

	return nil
}

func (s *Manager) createButch(rows []*wal.Row) ([][]byte, error) {
	batch := make([][]byte, len(rows))

	for i, row := range rows {

		rB, err := row.Marshal()
		if err != nil {
			s.log.Error("failed to marshal row", logger.Error(err))
			return nil, err
		}
		batch[i] = rB
	}
	return batch, nil
}

func (s *Manager) ReadSegment(name string) ([]byte, error) {
	if s.curSeg.GetName() == name {
		return s.readCurrSegment()
	} else {
		return s.readFile(name)
	}
}

func (s *Manager) readCurrSegment() ([]byte, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	data, err := s.curSeg.Read()
	if err != nil {
		s.log.Error("error reading from file", logger.Error(err))
		return nil, err
	}
	return data, nil
}

func (s *Manager) readFile(name string) ([]byte, error) {
	data, err := fs.ReadFile(name)
	if err != nil {
		s.log.Error("error opening file", logger.Error(err))
		return nil, err
	}

	return data, nil

}
