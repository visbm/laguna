package concurrency

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSemaphore_ConcurrentAcquireRelease(t *testing.T) {
	const max = 5
	const goroutines = 50

	sem := NewSemaphore(max)
	var active int32
	var maxActive int32
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem.Acquire()

			cur := atomic.AddInt32(&active, 1)
			if cur > int32(max) {
				t.Errorf("active goroutines exceeded limit: %d > %d", cur, max)
			}
			for {
				old := atomic.LoadInt32(&maxActive)
				if cur > old {
					if atomic.CompareAndSwapInt32(&maxActive, old, cur) {
						break
					}
				} else {
					break
				}
			}

			time.Sleep(time.Millisecond * 10)
			atomic.AddInt32(&active, -1)
			sem.Release()
		}()
	}

	wg.Wait()

	if maxActive != int32(max) {
		t.Errorf("expected max active == %d, got %d", max, maxActive)
	}
}

func TestSemaphore_AcquireWithDeadline(t *testing.T) {
	sem := NewSemaphore(1)

	sem.Acquire()

	start := time.Now()
	ok := sem.AcquireWithDeadline(50 * time.Millisecond)
	elapsed := time.Since(start)

	if ok {
		t.Errorf("expected AcquireWithDeadline to fail when full")
	}

	if elapsed < 45*time.Millisecond {
		t.Errorf("expected blocking close to deadline, got %v", elapsed)
	}

	sem.Release()

	ok = sem.AcquireWithDeadline(50 * time.Millisecond)
	if !ok {
		t.Errorf("expected AcquireWithDeadline to succeed after release")
	}
}
