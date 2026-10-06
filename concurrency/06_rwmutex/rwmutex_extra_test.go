package rwmutex

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWriterDoesNotStarve(t *testing.T) {
	var rw RWMutex

	stop := make(chan struct{})
	var readers sync.WaitGroup

	for range 8 {
		readers.Add(1)

		go func() {
			defer readers.Done()

			for {
				select {
				case <-stop:
					return
				default:
				}

				rw.RLock()
				time.Sleep(time.Millisecond)
				rw.RUnlock()
			}
		}()
	}

	time.Sleep(50 * time.Millisecond)

	acquired := make(chan struct{})

	go func() {
		rw.Lock()
		close(acquired)
		rw.Unlock()
	}()

	select {
	case <-acquired:
	case <-time.After(3 * time.Second):
		close(stop)
		readers.Wait()
		t.Fatal("writer starved")
	}

	close(stop)
	readers.Wait()
}

func TestReaderCanProceedAfterQueuedWriterFinishes(t *testing.T) {
	var rw RWMutex
	rw.RLock()

	writerDone := finished(func() {
		rw.Lock()
		rw.Unlock()
	})

	deadline := time.Now().Add(3 * time.Second)
	for atomic.LoadUint32(&rw.waitingWriters) == 0 {
		if time.Now().After(deadline) {
			rw.RUnlock()
			t.Fatal("writer did not join the wait queue")
		}
		time.Sleep(time.Millisecond)
	}
	currentWaitingWriters := atomic.LoadUint32(&rw.waitingWriters)

	rw.RUnlock()
	select {
	case <-writerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("writer did not finish")
	}

	rw.RLock()
	readerDone := finished(func() {
		rw.waitForWriters(currentWaitingWriters)
		rw.RLock()
		rw.RUnlock()
	})

	select {
	case <-readerDone:
		rw.RUnlock()
	case <-time.After(3 * time.Second):
		rw.RUnlock()
		select {
		case <-readerDone:
		case <-time.After(3 * time.Second):
			t.Fatal("reader did not wake after cleanup")
		}
		t.Fatal("reader waited for an unrelated reader after the writer finished")
	}
}

func TestQueuedReadersWakeAfterWriterAcquires(t *testing.T) {
	var rw RWMutex
	rw.RLock()

	writerAcquired := make(chan struct{})
	releaseWriter := make(chan struct{})
	writerDone := finished(func() {
		rw.Lock()
		close(writerAcquired)
		<-releaseWriter
		rw.Unlock()
	})

	deadline := time.Now().Add(3 * time.Second)
	for atomic.LoadUint32(&rw.waitingWriters) == 0 {
		if time.Now().After(deadline) {
			rw.RUnlock()
			close(releaseWriter)
			t.Fatal("writer did not join the wait queue")
		}
		time.Sleep(time.Millisecond)
	}

	readerDone := finished(func() {
		rw.RLock()
		rw.RUnlock()
	})

	deadline = time.Now().Add(3 * time.Second)
	for rw.readerWaitersCnt.Load() == 0 {
		if time.Now().After(deadline) {
			rw.RUnlock()
			close(releaseWriter)
			t.Fatal("reader did not join the writer wait queue")
		}
		time.Sleep(time.Millisecond)
	}

	rw.RUnlock()
	select {
	case <-writerAcquired:
	case <-time.After(3 * time.Second):
		close(releaseWriter)
		t.Fatal("writer did not acquire the lock")
	}

	select {
	case <-readerDone:
		close(releaseWriter)
		t.Fatal("reader acquired the lock while the writer held it")
	default:
	}

	close(releaseWriter)
	select {
	case <-writerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("writer did not finish")
	}
	select {
	case <-readerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("reader did not wake after the writer finished")
	}
}
