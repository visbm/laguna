package fs

import (
	"os"
	"strconv"
	"time"
)

type FileSegment struct {
	file    *os.File
	path    string
	size    int64
	maxSize int64
}

func NewFileSegment(path string, maxSize int64) (*FileSegment, error) {
	id := time.Now().UnixMicro()
	name := path + "/" + strconv.FormatInt(id, 10) + ".log"

	file, err := os.OpenFile(name, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &FileSegment{
		file:    file,
		path:    path,
		size:    0,
		maxSize: maxSize,
	}, nil
}

func (f *FileSegment) Rotate() error {
	err := f.file.Close()
	if err != nil {
		return err
	}

	id := time.Now().UnixMicro()
	name := f.path + "/" + strconv.FormatInt(id, 10) + ".log"

	file, err := os.OpenFile(name, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	f.file = file
	f.size = 0
	return nil
}

func (f *FileSegment) Write(data []byte) error {
	n, err := f.file.Write(data)
	if err != nil {
		return err
	}

	f.size += int64(n)
	return nil

}

func (f *FileSegment) Close() error {
	return f.file.Close()
}

func (f *FileSegment) Fits(fileSize int64) bool {
	return f.maxSize >= fileSize+f.size
}
