package semaphore

import "testing"

func TestNewNegativePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	New(-1)
}

func TestReleaseOverflowPanics(t *testing.T) {
	s := &Semaphore{permits: ^uint32(0)}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	s.Release()
}
