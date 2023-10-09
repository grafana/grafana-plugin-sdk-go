package instancemgmt

import (
	"sync"
	"time"
)

// instanceDisposer tracks and disposes of disposable instances.
type instanceDisposer struct {
	disposeTTL time.Duration
	cache      sync.Map
	m          sync.RWMutex
}

// newInstanceDisposer creates a new instanceDisposer.
func newInstanceDisposer(disposeTTL time.Duration) instanceDisposer {
	return instanceDisposer{
		disposeTTL: disposeTTL,
		cache:      sync.Map{},
	}
}

// disposable returns whether an instance is disposable.
func (d *instanceDisposer) disposable(i Instance) (InstanceDisposer, bool) {
	if id, ok := i.(InstanceDisposer); ok {
		return id, true
	}
	return nil, false
}

// track tracks a disposable instance. If there is a disposable instance already, it will be disposed of first.
func (d *instanceDisposer) track(cacheKey interface{}, id InstanceDisposer) {
	d.m.Lock()
	defer d.m.Unlock()

	// If there is a disposable instance already, we should dispose of it first.
	if i, exists := d.cache.LoadAndDelete(cacheKey); exists {
		i.(InstanceDisposer).Dispose()
		activeInstances.Dec()
	}

	d.cache.Store(cacheKey, id)
}

// tracking returns whether a disposable instance is being tracked.
func (d *instanceDisposer) tracking(cacheKey interface{}) bool {
	d.m.RLock()
	defer d.m.RUnlock()
	_, exists := d.cache.Load(cacheKey)
	return exists
}

// dispose disposes of a disposable instance.
func (d *instanceDisposer) dispose(cacheKey interface{}) {
	time.AfterFunc(d.disposeTTL, func() {
		d.m.Lock()
		defer d.m.Unlock()
		if i, ok := d.cache.LoadAndDelete(cacheKey); ok {
			i.(InstanceDisposer).Dispose()
			activeInstances.Dec()
		}
	})
}
