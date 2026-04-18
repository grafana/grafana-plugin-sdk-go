package instancemgmt

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
)

// benchmarkLocker exercises a pre-populated locker with a configurable
// number of distinct keys and a configurable write ratio (0..100).
// It's intended to measure the steady-state hot path where all keys are
// already present in the map and lookups dominate.
func benchmarkLocker(b *testing.B, numKeys int, writeRatio int) {
	b.Helper()
	lkr := newLocker()
	for i := range numKeys {
		lkr.RLock(i)
		lkr.RUnlock(i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewSource(rand.Int63())) //nolint:gosec // benchmark-only, not a security context
		for pb.Next() {
			key := r.Intn(numKeys)
			if writeRatio > 0 && r.Intn(100) < writeRatio {
				lkr.Lock(key)
				lkr.Unlock(key)
			} else {
				lkr.RLock(key)
				lkr.RUnlock(key)
			}
		}
	})
}

func BenchmarkLocker_RLockOnly(b *testing.B) {
	for _, keys := range []int{1, 10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("keys=%d", keys), func(b *testing.B) {
			benchmarkLocker(b, keys, 0)
		})
	}
}

func BenchmarkLocker_Mixed_90_10(b *testing.B) {
	for _, keys := range []int{1, 10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("keys=%d", keys), func(b *testing.B) {
			benchmarkLocker(b, keys, 10)
		})
	}
}

func BenchmarkLocker_Mixed_50_50(b *testing.B) {
	for _, keys := range []int{1, 10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("keys=%d", keys), func(b *testing.B) {
			benchmarkLocker(b, keys, 50)
		})
	}
}

// BenchmarkLocker_FirstAccess measures the cold path: every iteration
// uses a brand-new key, so every op pays the map-write cost.
func BenchmarkLocker_FirstAccess(b *testing.B) {
	lkr := newLocker()
	var counter int64

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := atomic.AddInt64(&counter, 1)
			lkr.RLock(key)
			lkr.RUnlock(key)
		}
	})
}
