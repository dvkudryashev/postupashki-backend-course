package rwmutex

import (
	"sync"
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
