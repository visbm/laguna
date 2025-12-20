package segment

import (
	"io"
	"laguna/common/logger"
	"laguna/utils/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type FileSegment struct {
	file *os.File

	path    string
	size    int64
	maxSize int64

	log logger.Logger
}

func NewFileSegment(log logger.Logger, path string, maxSize int64) (*FileSegment, error) {
	file, err := initFileSegment(path)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return &FileSegment{
		file:    file,
		path:    path,
		size:    stat.Size(),
		maxSize: maxSize,

		log: log,
	}, nil
}

func (f *FileSegment) Rotate() error {
	err := f.file.Close()
	if err != nil {
		f.log.Error("error closed file", logger.Error(err))
	}

	file, err := newFile(f.path)
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

	err = f.file.Sync()
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

func initFileSegment(path string) (*os.File, error) {
	names, err := fs.ReadDir(path)
	if err != nil {
		return nil, err
	}

	if len(names) == 0 {
		return newFile(path)
	}

	file, err := fs.OpenFile(path + "/" + names[len(names)-1])

	if err != nil {
		return nil, err
	}

	return file, nil
}

func newFile(path string) (*os.File, error) {
	id := time.Now().UnixMicro()
	name := path + "/" + strconv.FormatInt(id, 10) + ".bin"

	file, err := fs.OpenFile(name)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (f *FileSegment) GetName() string {
	return filepath.Base(f.file.Name())
}

func (f *FileSegment) CurrentOffset() int64 {
	return f.size
}

func (f *FileSegment) Read() ([]byte, error) {
	buf, err := io.ReadAll(f.file)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
