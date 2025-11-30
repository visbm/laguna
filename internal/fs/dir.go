package fs

import (
	"os"
)

func ReadDir(path string) ([]string, error) {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return nil, nil
	}
	names := make([]string, len(dirs))
	for i, dir := range dirs {
		names[i] = dir.Name()
	}
	return names, nil
}

func OpenFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return file, nil
}
