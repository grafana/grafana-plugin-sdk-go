package instancemgmt

import (
	"fmt"
	"sync"
)

type locker struct {
	locks   map[interface{}]*sync.RWMutex
	locksRW *sync.RWMutex
}

func newLocker() *locker {
	return &locker{
		locks:   make(map[interface{}]*sync.RWMutex),
		locksRW: new(sync.RWMutex),
	}
}

func (lkr *locker) Lock(key interface{}) {
	lk, ok := lkr.getLock(key)
	if !ok {
		lk = lkr.newLock(key)
	}
	lk.Lock()
}

func (lkr *locker) Unlock(key interface{}) {
	lk, ok := lkr.getLock(key)
	if !ok {
		panic(fmt.Errorf("lock for key '%s' not initialized", key))
	}
	lk.Unlock()
}

func (lkr *locker) RLock(key interface{}) {
	lk, ok := lkr.getLock(key)
	if !ok {
		lk = lkr.newLock(key)
	}
	lk.RLock()
}

func (lkr *locker) RUnlock(key interface{}) {
	lk, ok := lkr.getLock(key)
	if !ok {
		panic(fmt.Errorf("lock for key '%s' not initialized", key))
	}
	lk.RUnlock()
}

func (lkr *locker) newLock(key interface{}) *sync.RWMutex {
	lkr.locksRW.Lock()
	defer lkr.locksRW.Unlock()

	if lk, ok := lkr.locks[key]; ok {
		return lk
	}
	lk := new(sync.RWMutex)
	lkr.locks[key] = lk
	return lk
}

func (lkr *locker) getLock(key interface{}) (*sync.RWMutex, bool) {
	lkr.locksRW.RLock()
	defer lkr.locksRW.RUnlock()

	lock, ok := lkr.locks[key]
	return lock, ok
}
