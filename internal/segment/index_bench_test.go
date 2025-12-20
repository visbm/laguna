package segment

import (
	"testing"
)

// go test -bench . -benchmem

//choose depends on your type of load
// BenchmarkIndexManagerMutex_Add-12                7920044               130.2 ns/op           202 B/op          0 allocs/op
// BenchmarkIndexManagerMutex_AddBatch-12             89304             13375 ns/op               2 B/op          0 allocs/op
// BenchmarkIndexManagerMutex_Get-12               99791061                11.61 ns/op            0 B/op          0 allocs/op
// BenchmarkIndexManagerCOW_Add-12                    10000            109002 ns/op          410650 B/op         18 allocs/op
// BenchmarkIndexManagerCOW_AddBatch-12               27120             39769 ns/op          196813 B/op         10 allocs/op
// BenchmarkIndexManagerCOW_Get-12                 161146656                7.956 ns/op           0 B/op          0 allocs/op
// BenchmarkIndexManagerArray_Add-12               32393010                42.48 ns/op          190 B/op          0 allocs/op
// BenchmarkIndexManagerArray_AddBatch-12             57024             29273 ns/op          110311 B/op          0 allocs/op
// BenchmarkIndexManagerArray_Get-12               35283622                32.16 ns/op            0 B/op          0 allocs/op

const (
	N         = 100000
	batchSize = 1000
)

// =========================
// ======== Mutex ==========
// =========================

func BenchmarkIndexManagerMutex_Add(b *testing.B) {
	im := NewIndexManagerMutex()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}
}

func BenchmarkIndexManagerMutex_AddBatch(b *testing.B) {
	im := NewIndexManagerMutex()

	entries := make(map[uint64]IndexEntry)
	for i := 0; i < batchSize; i++ {
		entries[uint64(i)] = IndexEntry{FileName: "file.log", Offset: int64(i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im.AddBatch(entries)
	}
}

func BenchmarkIndexManagerMutex_Get(b *testing.B) {
	im := NewIndexManagerMutex()

	for i := 0; i < N; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = im.GetIndex(uint64(i % N))
	}
}

// =========================
// =========== COW =========
// =========================

func BenchmarkIndexManagerCOW_Add(b *testing.B) {
	im := NewIndexManagerCOW()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}
}

func BenchmarkIndexManagerCOW_AddBatch(b *testing.B) {
	im := NewIndexManagerCOW()

	entries := make(map[uint64]IndexEntry)
	for i := 0; i < batchSize; i++ {
		entries[uint64(i)] = IndexEntry{FileName: "file.log", Offset: int64(i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im.AddBatch(entries)
	}
}

func BenchmarkIndexManagerCOW_Get(b *testing.B) {
	im := NewIndexManagerCOW()

	for i := 0; i < N; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = im.GetIndex(uint64(i % N))
	}
}

// =========================
// ====== Array (sorted) ====
// =========================

func BenchmarkIndexManagerArray_Add(b *testing.B) {
	im := NewIndexManagerArray()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}
}

func BenchmarkIndexManagerArray_AddBatch(b *testing.B) {
	im := NewIndexManagerArray()
	im.entries = make([]IndexEntry, 0, 1_000_000_000)

	const batchSize = 1000
	entries := make([]IndexEntry, batchSize)
	for i := 0; i < batchSize; i++ {
		entries[i] = IndexEntry{
			LSN:      uint64(i),
			FileName: "file.log",
			Offset:   int64(i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im.AddBatch(entries)
	}
}

func BenchmarkIndexManagerArray_Get(b *testing.B) {
	im := NewIndexManagerArray()

	for i := 0; i < N; i++ {
		im.Add(uint64(i), "file.log", int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = im.GetIndex(uint64(i % N))
	}
}
