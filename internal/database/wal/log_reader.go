package wal

import (
	"fmt"

	"io"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/fs"
	"os"
)

type LogReader struct {
	log       logger.Logger
	directory string `yaml:"directory"`
}

func NewLogReader(conf config.WAL, log logger.Logger) *LogReader {
	return &LogReader{
		log:       log,
		directory: conf.Directory,
	}
}

func (lr *LogReader) Read() ([]*Row, error) {
	dirs, err := fs.ReadDir(lr.directory)
	if err != nil {
		return nil, err
	}

	if len(dirs) == 0 {
		return nil, nil
	}

	resp := make([]*Row, 0, len(dirs))
	for _, fileName := range dirs {

		file, err := fs.OpenFile(lr.directory + "/" + fileName)
		if err != nil {
			return nil, err
		}

		data, err := lr.readFile(file)

		resp = append(resp, data...)

		err = file.Close()
		if err != nil {
			lr.log.Error("non fatal error closing file", logger.Error(err))
		}
	}

	return resp, nil
}

func (lr *LogReader) readFile(f *os.File) ([]*Row, error) {
	var result []*Row
	buf := make([]byte, 4096)
	var leftover []byte
	for {
		n, err := f.Read(buf)
		if n > 0 {
			leftover = append(leftover, buf[:n]...)
			offset := 0

			for offset < len(leftover) {
				r := &Row{}
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
			return nil, fmt.Errorf("read error: %w", err)
		}
	}

	if len(leftover) > 0 {
		offset := 0
		for offset < len(leftover) {
			r := &Row{}
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
