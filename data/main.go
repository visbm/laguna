package main

import (
	"fmt"
	"io"
	"laguna/internal/database/wal"
	"os"
)

func main() {
	rows, err := ReadWALFile("/Users/nick/goSelfEducation/laguna/data/wal/1763469986037616.bin")
	if err != nil {
		fmt.Println("Error reading WAL file:", err)
		return
	}
	PrintRows(rows)
}

func ReadWALFile(filePath string) ([]wal.Row, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result []wal.Row
	buf := make([]byte, 4096)

	var leftover []byte

	for {
		n, err := f.Read(buf)
		if n > 0 {
			data := append(leftover, buf[:n]...)
			offset := 0
			for offset < len(data) {
				r := wal.Row{}
				readBytes, err := parseRow(data[offset:], &r)
				if err != nil {
					if err == io.EOF {
						leftover = data[offset:]
						break
					}
					return nil, err
				}
				result = append(result, r)
				offset += readBytes
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	if len(leftover) > 0 {
		offset := 0
		for offset < len(leftover) {
			r := &wal.Row{}
			readBytes, err := r.Unmarshal(leftover[offset:])
			if err != nil {
				break
			}
			result = append(result, *r)
			offset += readBytes
		}
	}
	return result, nil
}

func parseRow(data []byte, r *wal.Row) (int, error) {
	offset, err := r.Unmarshal(data)
	if err != nil {
		return 0, err
	}
	return offset, nil
}

func PrintRows(rows []wal.Row) {
	for _, r := range rows {
		fmt.Printf("%+v\n", r)
	}
}
