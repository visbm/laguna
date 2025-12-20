package inmemory

import (
	"fmt"
	"strconv"
	"testing"
)

//go test -bench=. -benchmem

func benchmarkInMemorySet(b *testing.B, shards int64) {
	m := NewInMemory(shards)

	b.Run(fmt.Sprintf("shards_%d", shards), func(b *testing.B) {
		b.SetParallelism(4)
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "key_" + strconv.Itoa(i)
				value := "value_" + strconv.Itoa(i)
				_ = m.Set(key, value)
				i++
			}
		})
	})
}

func benchmarkInMemoryGet(b *testing.B, shards int64) {
	m := NewInMemory(shards)

	for i := 0; i < 50000; i++ {
		_ = m.Set("key_"+strconv.Itoa(i), "value_"+strconv.Itoa(i))
	}

	b.Run(fmt.Sprintf("shards_%d", shards), func(b *testing.B) {
		b.SetParallelism(4)
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := "key_" + strconv.Itoa(i%50000)
				_, _ = m.Get(key)
				i++
			}
		})
	})
}

func BenchmarkInMemorySet(b *testing.B) {
	benchmarks := []int64{1, 4, 16, 32, 64, 128}
	for _, shards := range benchmarks {
		benchmarkInMemorySet(b, shards)
	}
}

func BenchmarkInMemoryGet(b *testing.B) {
	benchmarks := []int64{1, 4, 16, 32, 64, 128}
	for _, shards := range benchmarks {
		benchmarkInMemoryGet(b, shards)
	}
}
