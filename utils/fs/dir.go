package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ReadDir(path string) ([]string, error) {
	dirs, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
			return nil, nil
		}
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
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func ReadDirsFrom(path string, fileFrom string) ([]string, error) {
	dirs, err := ReadDir(path)
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return nil, nil
	}

	if fileFrom == "" {
		return dirs, nil
	}

	idx := binarySearch(dirs, fileFrom)

	if idx == -1 {
		return nil, nil
	}

	return dirs[idx:], nil
}

func binarySearch(array []string, target string) int {
	low, high := 0, len(array)-1

	for low <= high {
		mid := (low + high) / 2
		if array[mid] == target {
			return mid
		} else if array[mid] < target {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return -1
}
