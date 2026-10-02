package rwmutex

import (
	"primitives/internal/futex"
	"sync/atomic"
)

const (
	free   = 0
	writer = 1 << 31
)

type RWMutex struct {
	state          uint32
	waitingWriters atomic.Int32
	waitersCnt     atomic.Int32
}

func (rw *RWMutex) RLock() {
	for {
		currentState := atomic.LoadUint32(&rw.state)

		if currentState == writer {
			rw.waitersCnt.Add(1)
			futex.Wait(&rw.state, currentState)
			rw.waitersCnt.Add(-1)
			continue
		}

		if rw.waitingWriters.Load() > 0 {
			if currentState == free {
				continue
			}

			rw.waitersCnt.Add(1)
			futex.Wait(&rw.state, currentState)
			rw.waitersCnt.Add(-1)
			continue
		}

		if currentState == writer-1 {
			panic("reader count overflow")
		}

		if atomic.CompareAndSwapUint32(&rw.state, currentState, currentState+1) {
			return
		}
	}
}

func (rw *RWMutex) RUnlock() {
	for {
		currentState := atomic.LoadUint32(&rw.state)
		switch {
		case currentState == free:
			panic("tried to unlock unlocked")
		case currentState == writer:
			panic("tried to unlock writer's lock")
		case currentState > 0 && currentState < writer:
			if atomic.CompareAndSwapUint32(&rw.state, currentState, currentState-1) {
				if currentState > 1 {
					return
				}
				if currentState == 1 {
					if rw.waitersCnt.Load() > 0 {
						futex.WakeAll(&rw.state)
					}
					return
				}
			}
		default:
			panic("unexpected behavior")
		}
	}
}

func (rw *RWMutex) Lock() {
	enqueued := false
	for {
		currentState := atomic.LoadUint32(&rw.state)
		if currentState == free {
			if atomic.CompareAndSwapUint32(&rw.state, free, writer) {
				if enqueued {
					rw.waitingWriters.Add(-1)
				}
				return
			}
			if !enqueued {
				rw.waitingWriters.Add(1)
				enqueued = true
			}
			continue
		}
		if !enqueued {
			rw.waitingWriters.Add(1)
			enqueued = true
			continue
		}
		rw.waitersCnt.Add(1)
		futex.Wait(&rw.state, currentState)
		rw.waitersCnt.Add(-1)
	}
}

func (rw *RWMutex) Unlock() {
	if atomic.CompareAndSwapUint32(&rw.state, writer, free) {
		if rw.waitersCnt.Load() > 0 {
			futex.WakeAll(&rw.state)
		}
		return
	}
	panic("tried to unlock non-writer's lock")
}
