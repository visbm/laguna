package logs

import (
	"fmt"
	"laguna/utils/concurrency"

	"io"
	"laguna/common/logger"
	"laguna/internal/database/wal"
	"laguna/internal/fs"
)

type LogReader struct {
	log logger.Logger
}

func NewLogReader(log logger.Logger) *LogReader {
	return &LogReader{
		log: log,
	}
}

// ReadFrom todo if need
func (lr *LogReader) ReadFrom(directory string, target string) ([]*wal.Row, error) {
	names, err := fs.ReadDirsForm(directory, target)
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

func (lr *LogReader) ReadFromFilesStream(directory string) concurrency.FutureRespWithErr[[]*wal.Row] {
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

		lr.readFilesStream(directory, names, resp)
	}()

	return resp
}

func (lr *LogReader) readFilesStream(mainDir string, names []string, resp concurrency.FutureRespWithErr[[]*wal.Row]) {
	for _, fileName := range names {

		file, err := fs.OpenFile(mainDir + "/" + fileName)
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

		err = file.Close()
		if err != nil {
			lr.log.Error("non fatal error closing file", logger.Error(err))
		}
	}
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

func (lr *LogReader) Read(r io.Reader) ([]*wal.Row, error) {
	var result []*wal.Row
	buf := make([]byte, 4096)
	var leftover []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			leftover = append(leftover, buf[:n]...)
			offset := 0

			for offset < len(leftover) {
				r := &wal.Row{}
				readBytes, err := r.Unmarshal(leftover[offset:])
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
				result = append(result, r)
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
