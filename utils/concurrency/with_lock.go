package concurrency

import "sync"

func WithLock(m *sync.Mutex, fn func()) {
	if fn == nil || m == nil {
		return
	}

	m.Lock()
	defer m.Unlock()
	fn()
}
