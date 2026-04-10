package storage

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAddReturnsSameInstance directly tests add() - this reliably catches the bug
// because it bypasses get() and forces all goroutines through add().
func TestAddReturnsSameInstance(t *testing.T) {
	testFiles := files{files: map[string]*file{}}
	testPath := "/test/add/instance.har"

	const numGoroutines = 100

	results := make(chan *file, numGoroutines)
	ready := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ready
			f := testFiles.getOrAdd(testPath) // Call add() directly
			results <- f
		}()
	}

	close(ready)
	wg.Wait()
	close(results)

	var allFiles []*file
	for f := range results {
		allFiles = append(allFiles, f)
	}

	first := allFiles[0]
	for i, f := range allFiles {
		require.Same(t, first, f, "goroutine %d got different *file instance (got %p, want %p)", i, f, first)
	}
}

// TestConcurrentLockUnlockPanic verifies that concurrent rLock/rUnlock calls don't panic.
// A panic like "sync: RUnlock of unlocked RWMutex" indicates that rLock() and rUnlock()
// got different *file instances due to a race in getOrAdd().
func TestConcurrentLockUnlockPanic(t *testing.T) {
	for run := 0; run < 10; run++ {
		testFiles := files{files: map[string]*file{}}
		testPath := "/test/panic/repro.har"

		const numGoroutines = 100

		var wg sync.WaitGroup
		ready := make(chan struct{})
		panics := make(chan any, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panics <- r
					}
				}()
				<-ready
				testFiles.rLock(testPath)
				testFiles.rUnlock(testPath)
			}()
		}

		close(ready)
		wg.Wait()
		close(panics)

		// Assert no panics occurred
		for p := range panics {
			require.Fail(t, "goroutine panicked", "panic: %v", p)
		}
	}
}
