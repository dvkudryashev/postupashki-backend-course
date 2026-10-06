package rwmutex

import (
	"primitives/internal/futex"
	"sync/atomic"
)

const (
	free   uint32 = 0
	writer uint32 = 1 << 31
)

type RWMutex struct {
	state            uint32
	waitingWriters   uint32
	waitersCnt       atomic.Int32
	readerWaitersCnt atomic.Int32
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

		currentWaitingWriters := atomic.LoadUint32(&rw.waitingWriters)
		if currentWaitingWriters > 0 {
			rw.waitForWriters(currentWaitingWriters)
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

func (rw *RWMutex) waitForWriters(currentWaitingWriters uint32) {
	rw.readerWaitersCnt.Add(1)
	futex.Wait(&rw.waitingWriters, currentWaitingWriters)
	rw.readerWaitersCnt.Add(-1)
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
				if currentState == 1 && rw.waitersCnt.Load() > 0 {
					futex.WakeAll(&rw.state)
				}
				return
			}
		default:
			panic("unexpected rwmutex state")
		}
	}
}

func (rw *RWMutex) Lock() {
	if atomic.CompareAndSwapUint32(&rw.state, free, writer) {
		return
	}

	atomic.AddUint32(&rw.waitingWriters, 1)

	for {
		currentState := atomic.LoadUint32(&rw.state)
		if currentState == free {
			if atomic.CompareAndSwapUint32(&rw.state, free, writer) {
				if atomic.AddUint32(&rw.waitingWriters, ^uint32(0)) == 0 && rw.readerWaitersCnt.Load() > 0 {
					futex.WakeAll(&rw.waitingWriters)
				}
				return
			}
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
