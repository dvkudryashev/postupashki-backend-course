package mutex

import (
	"primitives/internal/futex"
	"sync/atomic"
)

const (
	free uint32 = iota
	held
	contended
)

type Mutex struct {
	state uint32
}

func (m *Mutex) Lock() {
	if atomic.CompareAndSwapUint32(&m.state, free, held) {
		return
	}

	for atomic.SwapUint32(&m.state, contended) != free {
		futex.Wait(&m.state, contended)
	}
}

func (m *Mutex) TryLock() bool {
	return atomic.CompareAndSwapUint32(&m.state, free, held)
}

func (m *Mutex) Unlock() {
	switch atomic.SwapUint32(&m.state, free) {
	case free:
		panic("tried to unlock free mutex")
	case held:
		return
	case contended:
		futex.Wake(&m.state)
	default:
		panic("unexpected mutex state")
	}
}
