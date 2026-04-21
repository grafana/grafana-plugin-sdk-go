package instancemgmt

import (
	"fmt"
	"sync"
)

// locker is a named reader/writer mutual exclusion lock.
// The lock for each particular key can be held by an arbitrary number of readers or a single writer.
type locker struct {
	locks sync.Map
}

func newLocker() *locker {
	return &locker{}
}

// Lock locks named rw mutex with specified key for writing.
// If the lock with the same key is already locked for reading or writing,
// Lock blocks until the lock is available.
func (lkr *locker) Lock(key interface{}) {
	lkr.loadOrStore(key).Lock()
}

// Unlock unlocks named rw mutex with specified key for writing. It is a run-time error if rw is
// not locked for writing on entry to Unlock.
func (lkr *locker) Unlock(key interface{}) {
	lk, ok := lkr.load(key)
	if !ok {
		panic(fmt.Errorf("lock for key '%s' not initialized", key))
	}
	lk.Unlock()
}

// RLock locks named rw mutex with specified key for reading.
//
// It should not be used for recursive read locking for the same key; a blocked Lock
// call excludes new readers from acquiring the lock. See the
// documentation on the golang RWMutex type.
func (lkr *locker) RLock(key interface{}) {
	lkr.loadOrStore(key).RLock()
}

// RUnlock undoes a single RLock call for specified key;
// it does not affect other simultaneous readers of locker for specified key.
// It is a run-time error if locker for specified key is not locked for reading
func (lkr *locker) RUnlock(key interface{}) {
	lk, ok := lkr.load(key)
	if !ok {
		panic(fmt.Errorf("lock for key '%s' not initialized", key))
	}
	lk.RUnlock()
}

func (lkr *locker) load(key interface{}) (*sync.RWMutex, bool) {
	v, ok := lkr.locks.Load(key)
	if !ok {
		return nil, false
	}
	return v.(*sync.RWMutex), true
}

// loadOrStore returns the *sync.RWMutex for key, creating one if absent.
// On a creation race, LoadOrStore ensures every caller for a given key
// receives the same *sync.RWMutex; the losing mutex is discarded unlocked.
func (lkr *locker) loadOrStore(key interface{}) *sync.RWMutex {
	if v, ok := lkr.locks.Load(key); ok {
		return v.(*sync.RWMutex)
	}
	actual, _ := lkr.locks.LoadOrStore(key, &sync.RWMutex{})
	return actual.(*sync.RWMutex)
}
