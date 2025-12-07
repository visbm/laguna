package fs

import (
	"testing"
)

// go test -bench . -benchmem

//BenchmarkIndexManagerMutex_Add-12               12623743               103.3 ns/op           106 B/op          0 allocs/op
//BenchmarkIndexManagerMutex_AddBatch-12          18417960               118.1 ns/op           145 B/op          0 allocs/op
//BenchmarkIndexManagerMutex_Get-12               89259978                13.30 ns/op            0 B/op          0 allocs/op
//BenchmarkIndexManagerCOW_Add-12                    12524            120284 ns/op          405531 B/op         21 allocs/op
//BenchmarkIndexManagerCOW_AddBatch-12            19345465                75.70 ns/op           69 B/op          0 allocs/op
//BenchmarkIndexManagerCOW_Get-12                 15762070                75.75 ns/op           47 B/op          2 allocs/op

// Threshold coefficient W/R ≈ 2.63:
// If writes are more than ~2.6 times reads, COW (batch) is faster.
// Otherwise RWMutex is more efficient.

// RWMutex
func BenchmarkIndexManagerMutex_Add(b *testing.B) {
	im := NewIndexManagerMutex()
	for i := 0; i < b.N; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}
}

func BenchmarkIndexManagerMutex_AddBatch(b *testing.B) {
	im := NewIndexManagerMutex()
	entries := make(map[uint64]IndexEntry)
	for i := 0; i < b.N; i++ {
		entries[uint64(i)] = IndexEntry{FileName: "file.log", Offset: int64(i)}
	}
	b.ResetTimer()
	im.AddBatch(entries)
}

func BenchmarkIndexManagerMutex_Get(b *testing.B) {
	im := NewIndexManagerMutex()
	for i := 0; i < 100000; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = im.Get(uint64(i % 100000))
	}
}

// COW
func BenchmarkIndexManagerCOW_Add(b *testing.B) {
	im := NewIndexManagerCOW()
	for i := 0; i < b.N; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}
}

func BenchmarkIndexManagerCOW_AddBatch(b *testing.B) {
	im := NewIndexManagerCOW()
	entries := make(map[uint64]IndexEntry)
	for i := 0; i < b.N; i++ {
		entries[uint64(i)] = IndexEntry{FileName: "file.log", Offset: int64(i)}
	}
	b.ResetTimer()
	im.AddBatch(entries)
}

func BenchmarkIndexManagerCOW_Get(b *testing.B) {
	im := NewIndexManagerCOW()
	for i := 0; i < 1000; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = im.Get(uint64(i % 100000))
	}
}
