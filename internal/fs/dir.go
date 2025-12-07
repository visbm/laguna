package fs

import (
	"os"
	"path/filepath"
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

func ReadDirsForm(path string, fileForm string) ([]string, error) {
	dirs, err := ReadDir(path)
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return nil, nil
	}

	idx := upperBound(dirs, fileForm)

	return dirs[idx:], nil
}

func upperBound(array []string, target string) int {
	low, high := 0, len(array)-1

	for low <= high {
		mid := (low + high) / 2
		if array[mid] > target {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	return low
}
