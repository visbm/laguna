package logs

import (
	"bytes"
	"fmt"
	"io"
	"laguna/common/logger"
	"laguna/internal/database/wal"
	"laguna/internal/fs"
	"laguna/utils/concurrency"
)

type SegmentManager interface {
	ReadrSegment(name string) ([]byte, error)
	AddBatchIndex(entries map[uint64]fs.IndexEntry)
}

type LogReader struct {
	log logger.Logger
	sm  SegmentManager
}

func NewLogReader(sm SegmentManager, log logger.Logger) *LogReader {
	return &LogReader{
		log: log,
		sm:  sm,
	}
}

// ReadFrom  reads first file from offset
func (lr *LogReader) ReadFrom(directory string, target string, firstOffset int64) ([]*wal.Row, error) {

	var results []*wal.Row

	names, err := fs.ReadDirsFrom(directory, target)
	if err != nil {
		lr.log.Error("error read dirs", logger.Error(err))
		return results, err
	}

	if len(names) == 0 {
		return nil, nil
	}

	for i, name := range names {
		offset := int64(0)
		if i == 0 {
			offset = firstOffset
		}

		data, err := lr.sm.ReadrSegment(directory + "/" + name)
		if err != nil {
			lr.log.Error("error open file", logger.Error(err))
			return nil, err
		}

		buf := bytes.NewReader(data)
		rows, err := lr.readFromOffset(buf, offset)
		if err != nil {
			lr.log.Error("error readFromOffset", logger.Error(err))
			return nil, err
		}

		results = append(results, rows...)
	}

	return results, nil
}

func (lr *LogReader) ReadFromFiles(directory string) ([]*wal.Row, error) {
	names, err := fs.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	if len(names) == 0 {
		return nil, nil
	}

	resp, err := lr.readFiles(directory, names)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (lr *LogReader) RestoreSystemStream(directory string) concurrency.FutureRespWithErr[[]*wal.Row] {
	resp := concurrency.NewFutureRespWithErr[[]*wal.Row]()

	go func() {
		defer resp.Done()

		names, err := fs.ReadDir(directory)
		if err != nil {
			resp.Put(nil, err)
			return
		}

		if len(names) == 0 {
			return
		}

		for i, fileName := range names {

			file, err := fs.OpenFile(directory + "/" + fileName)
			if err != nil {
				resp.Put(nil, err)
				return
			}

			data, err := lr.Read(file)
			if err != nil {
				resp.Put(nil, err)
				return
			}

			resp.Put(data, nil)

			lr.restoreIndex(data, names[i])

			err = file.Close()
			if err != nil {
				lr.log.Error("non fatal error closing file", logger.Error(err))
			}
		}

	}()

	return resp
}

func (lr *LogReader) readFiles(mainDir string, names []string) ([]*wal.Row, error) {
	resp := make([]*wal.Row, 0, len(names))
	for _, fileName := range names {

		file, err := fs.OpenFile(mainDir + "/" + fileName)
		if err != nil {
			return nil, err
		}

		data, err := lr.Read(file)
		if err != nil {
			return nil, err
		}

		resp = append(resp, data...)

		err = file.Close()
		if err != nil {
			lr.log.Error("non fatal error closing file", logger.Error(err))
		}
	}
	return resp, nil
}

func (lr *LogReader) readFromOffset(r io.ReadSeeker, offset int64) ([]*wal.Row, error) {

	_, err := r.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	return lr.read(r)
}

func (lr *LogReader) Read(r io.Reader) ([]*wal.Row, error) {
	return lr.read(r)
}

func (lr *LogReader) read(r io.Reader) ([]*wal.Row, error) {
	var result []*wal.Row
	buf := make([]byte, 4096)
	var leftover []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			leftover = append(leftover, buf[:n]...)
			offset := 0

			for offset < len(leftover) {
				row := &wal.Row{}
				readBytes, err := row.Unmarshal(leftover[offset:])
				if err != nil {
					if err == io.EOF {
						break
					}
					lr.log.Warn("skipping corrupted row",
						logger.Integer("offset", int64(offset)),
						logger.Error(err),
					)
					offset++
					continue
				}
				result = append(result, row)
				offset += readBytes
			}

			leftover = leftover[offset:]
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("readFrom error: %w", err)
		}
	}

	if len(leftover) > 0 {
		offset := 0
		for offset < len(leftover) {
			r := &wal.Row{}
			readBytes, err := r.Unmarshal(leftover[offset:])
			if err != nil {
				lr.log.Warn("skipping final corrupted tail",
					logger.Integer("offset", int64(offset)),
					logger.Error(err),
				)
				break
			}
			result = append(result, r)
			offset += readBytes
		}
	}

	return result, nil
}

func (lr *LogReader) ReadStream(r io.Reader) concurrency.FutureRespWithErr[[]*wal.Row] {
	resp := concurrency.NewFutureRespWithErr[[]*wal.Row]()

	go func() {
		rows := make([]*wal.Row, 0)

		defer resp.Done()

		buf := make([]byte, 4096)
		var leftover []byte
		for {
			n, err := r.Read(buf)
			if n > 0 {
				leftover = append(leftover, buf[:n]...)
				offset := 0
				for offset < len(leftover) {
					row := &wal.Row{}
					readBytes, err := row.Unmarshal(leftover[offset:])
					if err != nil {
						if err == io.EOF {
							break
						}
						lr.log.Warn("skipping corrupted row",
							logger.Integer("offset", int64(offset)),
							logger.Error(err),
						)
						offset++
						continue
					}
					rows = append(rows, row)
					offset += readBytes
				}
				leftover = leftover[offset:]
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				lr.log.Error("read error", logger.Error(err))
				resp.Put(rows, err)
				return
			}
		}
		resp.Put(rows, nil)
	}()
	return resp
}

func (lr *LogReader) restoreIndex(rows []*wal.Row, fileName string) {
	offset := 0

	entries := make(map[uint64]fs.IndexEntry, len(rows))
	for _, row := range rows {
		b, err := row.Marshal()
		if err != nil {
			lr.log.Warn("skipping corrupted row", logger.Error(err))
			continue
		}

		entries[row.GetLsnID()] = fs.IndexEntry{FileName: fileName, Offset: int64(offset)}
		offset += len(b)
	}
	lr.sm.AddBatchIndex(entries)
	return
}
