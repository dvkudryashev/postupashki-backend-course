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

func waitForRWQueue(t *testing.T, rw *RWMutex, readers, writers uint32) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for {
		rw.guard.Lock()
		currentReaders := rw.waitingReaders
		currentWriters := rw.waitingWriters
		rw.guard.Unlock()

		if currentReaders == readers && currentWriters == writers {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("wait queue: readers=%d, writers=%d; expected readers=%d, writers=%d", currentReaders, currentWriters, readers, writers)
		}
		time.Sleep(time.Millisecond)
	}
}

func waitForRWEvent(t *testing.T, events <-chan string) string {
	t.Helper()

	select {
	case event := <-events:
		return event
	case <-time.After(3 * time.Second):
		t.Fatal("no lock acquisition event")
		return ""
	}
}

func waitForRWDone(t *testing.T, done <-chan struct{}, message string) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal(message)
	}
}

func TestWaitingReadersRunBeforeNextWriter(t *testing.T) {
	var rw RWMutex
	const readers = 3
	const writers = 2

	events := make(chan string, readers+writers)
	releaseReaders := make(chan struct{})
	releaseWriters := make(chan struct{})
	var readersRelease, writersRelease sync.Once
	var workers sync.WaitGroup

	stopReaders := func() {
		readersRelease.Do(func() { close(releaseReaders) })
	}
	stopWriters := func() {
		writersRelease.Do(func() { close(releaseWriters) })
	}

	rw.Lock()
	firstWriterHeld := true

	t.Cleanup(func() {
		if firstWriterHeld {
			rw.Unlock()
		}
		stopReaders()
		stopWriters()

		select {
		case <-finished(workers.Wait):
		case <-time.After(3 * time.Second):
			t.Error("workers did not finish after cleanup")
		}
	})

	for range writers {
		workers.Add(1)
		go func() {
			defer workers.Done()

			rw.Lock()
			events <- "writer"
			<-releaseWriters
			rw.Unlock()
		}()
	}
	waitForRWQueue(t, &rw, 0, writers)

	for range readers {
		workers.Add(1)
		go func() {
			defer workers.Done()

			rw.RLock()
			events <- "reader"
			<-releaseReaders
			rw.RUnlock()
		}()
	}
	waitForRWQueue(t, &rw, readers, writers)

	rw.Unlock()
	firstWriterHeld = false

	for range readers {
		if event := waitForRWEvent(t, events); event != "reader" {
			t.Fatal("writer acquired the lock before the waiting reader batch")
		}
	}

	select {
	case event := <-events:
		t.Fatalf("unexpected acquisition while readers hold the lock: %s", event)
	default:
	}

	stopReaders()
	if event := waitForRWEvent(t, events); event != "writer" {
		t.Fatalf("expected writer after the reader batch, got %s", event)
	}

	stopWriters()
	waitForRWDone(t, finished(workers.Wait), "workers did not finish")
}

func TestGrantedReaderCannotBeOvertaken(t *testing.T) {
	var rw RWMutex

	rw.Lock()
	firstWriterHeld := true
	readerRegistered := false

	writerAcquired := make(chan struct{})
	writerDone := make(chan struct{})
	releaseWriter := make(chan struct{})
	var writerRelease sync.Once

	stopWriter := func() {
		writerRelease.Do(func() { close(releaseWriter) })
	}

	t.Cleanup(func() {
		if firstWriterHeld {
			rw.Unlock()
		}
		stopWriter()

		if readerRegistered {
			rw.guard.Lock()
			currentState := atomic.LoadUint32(&rw.state)
			readerGranted := currentState > free && currentState < writer
			rw.guard.Unlock()

			if readerGranted {
				rw.RUnlock()
			}
		}

		select {
		case <-writerDone:
		case <-time.After(3 * time.Second):
			t.Error("writer did not finish after cleanup")
		}
	})

	go func() {
		rw.Lock()
		close(writerAcquired)
		<-releaseWriter
		rw.Unlock()
		close(writerDone)
	}()
	waitForRWQueue(t, &rw, 0, 1)

	currentRound, shouldWait := rw.registerReader()
	readerRegistered = true
	if !shouldWait {
		t.Fatal("reader was admitted while the first writer held the lock")
	}
	waitForRWQueue(t, &rw, 1, 1)

	rw.Unlock()
	firstWriterHeld = false

	rw.guard.Lock()
	currentState := atomic.LoadUint32(&rw.state)
	rw.guard.Unlock()

	if currentState != 1 {
		t.Fatalf("reader slot was not reserved: state=%d", currentState)
	}

	waitForRWDone(t, finished(func() {
		rw.waitForReaders(currentRound)
	}), "reader missed its grant before entering futex.Wait")

	select {
	case <-writerAcquired:
		t.Fatal("writer overtook the granted reader")
	default:
	}

	rw.RUnlock()
	readerRegistered = false

	waitForRWDone(t, writerAcquired, "writer did not acquire the lock after RUnlock")
	stopWriter()
	waitForRWDone(t, writerDone, "writer did not finish")
}

func TestReaderDoesNotStarve(t *testing.T) {
	var rw RWMutex

	stop := make(chan struct{})
	var writers sync.WaitGroup

	for range 4 {
		writers.Add(1)

		go func() {
			defer writers.Done()

			for {
				select {
				case <-stop:
					return
				default:
				}

				rw.Lock()
				time.Sleep(time.Microsecond * 200)
				rw.Unlock()
			}
		}()
	}

	time.Sleep(20 * time.Millisecond)

	readerDone := make(chan struct{})
	go func() {
		rw.RLock()
		rw.RUnlock()
		close(readerDone)
	}()

	readerStarved := false

	select {
	case <-readerDone:
	case <-time.After(3 * time.Second):
		readerStarved = true
	}

	close(stop)

	writersDone := make(chan struct{})

	go func() {
		writers.Wait()
		close(writersDone)
	}()

	select {
	case <-writersDone:
	case <-time.After(3 * time.Second):
		t.Fatal("writers did not stop")
	}

	select {
	case <-readerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("reader did not finish after writers stopped")
	}

	if readerStarved {
		t.Fatal("reader starved")
	}
}
