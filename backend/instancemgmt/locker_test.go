package instancemgmt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLocker(t *testing.T) {
	if testing.Short() {
		t.Skip("Tests with Sleep")
	}
	const notUpdated = "not_updated"
	const atThread1 = "at_thread_1"
	const atThread2 = "at_thread_2"
	t.Run("Should lock for same keys", func(t *testing.T) {
		updated := notUpdated
		locker := newLocker()
		locker.Lock(1)
		defer locker.Unlock(1)
		go func() {
			locker.RLock(1)
			defer locker.RUnlock(1)
			require.Equal(t, atThread1, updated, "Value should be updated in different thread")
			updated = atThread2
		}()
		time.Sleep(time.Millisecond * 10)
		require.Equal(t, notUpdated, updated, "Value should not be updated in different thread")
		updated = atThread1
	})

	t.Run("Should not lock for different keys", func(t *testing.T) {
		updated := notUpdated
		locker := newLocker()
		locker.Lock(1)
		defer locker.Unlock(1)
		go func() {
			locker.RLock(2)
			defer locker.RUnlock(2)
			require.Equal(t, notUpdated, updated, "Value should not be updated in different thread")
			updated = atThread2
		}()
		time.Sleep(time.Millisecond * 10)
		require.Equal(t, atThread2, updated, "Value should be updated in different thread")
		updated = atThread1
	})
}
