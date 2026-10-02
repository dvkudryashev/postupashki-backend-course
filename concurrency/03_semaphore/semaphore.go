package semaphore

import (
	"primitives/internal/futex"
	"sync/atomic"
)

type Semaphore struct {
	permits    uint32
	waitersCnt atomic.Int32
}

func New(n int) *Semaphore {
	if n < 0 {
		panic("negative permits")
	}

	if int64(n) > int64(^uint32(0)) {
		panic("uint32 overflow")
	}

	return &Semaphore{
		permits: uint32(n),
	}
}

func (s *Semaphore) Acquire() {
	for {
		permits := atomic.LoadUint32(&s.permits)
		if permits > 0 {
			if !atomic.CompareAndSwapUint32(&s.permits, permits, permits-1) {
				continue
			}
			return
		}
		s.waitersCnt.Add(1)
		futex.Wait(&s.permits, 0)
		s.waitersCnt.Add(-1)
	}
}

func (s *Semaphore) TryAcquire() bool {
	for {
		permits := atomic.LoadUint32(&s.permits)
		if permits == 0 {
			return false
		}
		if atomic.CompareAndSwapUint32(&s.permits, permits, permits-1) {
			return true
		}
	}
}

func (s *Semaphore) Release() {
	for {
		current := atomic.LoadUint32(&s.permits)
		newValue := int64(current) + 1

		if newValue > int64(^uint32(0)) {
			panic("uint32 overflow")
		}

		if atomic.CompareAndSwapUint32(&s.permits, current, uint32(newValue)) {
			if s.waitersCnt.Load() > 0 {
				futex.Wake(&s.permits)
			}
			return
		}
	}
}

func (s *Semaphore) Available() int {
	return int(atomic.LoadUint32(&s.permits))
}
