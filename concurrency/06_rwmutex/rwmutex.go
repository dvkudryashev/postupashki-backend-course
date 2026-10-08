package rwmutex

import (
	"primitives/02_mutex"
	"primitives/internal/futex"
	"sync/atomic"
)

const (
	free   uint32 = 0
	writer uint32 = 1 << 31
)

const maxReaders uint32 = writer - 1

type RWMutex struct {
	state          uint32
	guard          mutex.Mutex
	waitingReaders uint32
	waitingWriters uint32
	readerRound    uint32
}

func (rw *RWMutex) registerReader() (currentRound uint32, shouldWait bool) {
	rw.guard.Lock()
	defer rw.guard.Unlock()

	currentState := atomic.LoadUint32(&rw.state)
	if currentState != writer && rw.waitingWriters == 0 {
		if currentState >= maxReaders {
			panic("reader count overflow")
		}

		atomic.StoreUint32(&rw.state, currentState+1)
		return 0, false
	}

	if rw.waitingReaders >= maxReaders {
		panic("waiting reader count overflow")
	}

	currentRound = atomic.LoadUint32(&rw.readerRound)
	rw.waitingReaders++
	return currentRound, true
}

func (rw *RWMutex) RLock() {
	currentRound, shouldWait := rw.registerReader()
	if shouldWait {
		rw.waitForReaders(currentRound)
	}
}

func (rw *RWMutex) waitForReaders(currentRound uint32) {
	for atomic.LoadUint32(&rw.readerRound) == currentRound {
		futex.Wait(&rw.readerRound, currentRound)
	}
}

func (rw *RWMutex) RUnlock() {
	rw.guard.Lock()

	currentState := atomic.LoadUint32(&rw.state)
	if currentState >= writer {
		rw.guard.Unlock()
		panic("tried to unlock writer's lock")
	}
	if currentState == free {
		rw.guard.Unlock()
		panic("tried to unlock unlocked")
	}

	newValue := currentState - 1
	atomic.StoreUint32(&rw.state, newValue)
	wakeWriter := newValue == free && rw.waitingWriters > 0

	rw.guard.Unlock()

	if wakeWriter {
		futex.Wake(&rw.state)
	}
}

func (rw *RWMutex) Lock() {
	rw.guard.Lock()

	if rw.waitingWriters == ^uint32(0) {
		rw.guard.Unlock()
		panic("waiting writer count overflow")
	}
	rw.waitingWriters++

	currentState := atomic.LoadUint32(&rw.state)
	for currentState != free {
		rw.guard.Unlock()
		futex.Wait(&rw.state, currentState)
		rw.guard.Lock()
		currentState = atomic.LoadUint32(&rw.state)
	}

	rw.waitingWriters--
	atomic.StoreUint32(&rw.state, writer)

	rw.guard.Unlock()
}

func (rw *RWMutex) Unlock() {
	rw.guard.Lock()

	if atomic.LoadUint32(&rw.state) != writer {
		rw.guard.Unlock()
		panic("tried to unlock non-writer's lock")
	}

	readersToWake := rw.waitingReaders
	rw.waitingReaders = 0
	atomic.StoreUint32(&rw.state, readersToWake)
	wakeWriter := readersToWake == 0 && rw.waitingWriters > 0

	if readersToWake > 0 {
		atomic.AddUint32(&rw.readerRound, 1)
	}

	rw.guard.Unlock()

	if readersToWake > 0 {
		futex.WakeAll(&rw.readerRound)
	} else if wakeWriter {
		futex.Wake(&rw.state)
	}
}
