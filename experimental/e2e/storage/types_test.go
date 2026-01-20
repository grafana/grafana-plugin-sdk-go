package storage

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

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
			f := testFiles.add(testPath)
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
