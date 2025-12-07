package concurrency

import "time"

type Semaphore struct {
	ch chan struct{}
}

func NewSemaphore(max int64) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, max),
	}
}

func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

func (s *Semaphore) Release() {
	select {
	case <-s.ch:
	default:
		return
	}

}

func (s *Semaphore) AcquireWithDeadline(deadline time.Duration) bool {
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	select {
	case s.ch <- struct{}{}:
		timer.Stop()
		return true
	case <-timer.C:
		timer.Stop()
		return false
	}

}
