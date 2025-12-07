package id_generator

import (
	"sync"
	"testing"
)

func Test_NewIDGen_Concurrent(t *testing.T) {
	const goroutines = 100
	const perGoroutine = 1000

	tx := NewIDGenerator()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make(chan uint64, goroutines*perGoroutine)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				id := tx.NextID()
				results <- id
			}
		}()
	}

	wg.Wait()
	close(results)

	seen := make(map[uint64]bool)
	for id := range results {
		if seen[id] {
			t.Fatalf("duplicate id detected: %d", id)
		}
		seen[id] = true
	}

	expectedTotal := uint64(goroutines * perGoroutine)
	if len(seen) != int(expectedTotal) {
		t.Fatalf("expected %d unique IDs, got %d", expectedTotal, len(seen))
	}

	last := tx.id.Load()
	if last != expectedTotal {
		t.Fatalf("expected last id %d, got %d", expectedTotal, last)
	}
}

func Test_NewIDGenStart_Concurrent(t *testing.T) {
	const (
		goroutines   = 100
		perGoroutine = 1000
		start        = 100
	)

	tx := NewIDGeneratorWithStart(start)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make(chan uint64, goroutines*perGoroutine)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				id := tx.NextID()
				results <- id
			}
		}()
	}

	wg.Wait()
	close(results)

	seen := make(map[uint64]bool)
	for id := range results {
		if seen[id] {
			t.Fatalf("duplicate id detected: %d", id)
		}
		seen[id] = true
	}
	expectedTotal := uint64(goroutines * perGoroutine)
	if len(seen) != int(expectedTotal) {
		t.Fatalf("expected %d unique IDs, got %d", expectedTotal, len(seen))
	}

	expectedLast := uint64(goroutines*perGoroutine + start)
	last := tx.id.Load()
	if last != expectedLast {
		t.Fatalf("expected last id %d, got %d", expectedLast, last)
	}
}
