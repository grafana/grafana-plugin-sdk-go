package instancemgmt

import (
	"context"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// InstanceAcquirer is optionally implemented by instance managers that can keep
// an instance alive until a request has finished. Release must be called, even
// when the request fails or is canceled; canceling a context alone does not mean
// the handler has stopped using the instance.
type InstanceAcquirer interface {
	Acquire(ctx context.Context, pluginContext backend.PluginContext) (Instance, func(), error)
}

// Acquire obtains an instance and a release function. Managers implementing
// InstanceAcquirer defer disposal until release is called. Other managers retain
// their existing Get behavior and return a no-op release function.
func Acquire(ctx context.Context, manager InstanceManager, pluginContext backend.PluginContext) (Instance, func(), error) {
	if acquirer, ok := manager.(InstanceAcquirer); ok {
		return acquirer.Acquire(ctx, pluginContext)
	}
	instance, err := manager.Get(ctx, pluginContext)
	return instance, func() {}, err
}

type managedInstance struct {
	instance Instance
	mu       sync.Mutex
	users    int
	retired  bool
	ready    bool
	disposed bool
}

func (mi *managedInstance) acquire() (func(), bool) {
	mi.mu.Lock()
	defer mi.mu.Unlock()
	if mi.retired {
		return nil, false
	}
	mi.users++
	var once sync.Once
	return func() {
		once.Do(func() {
			mi.mu.Lock()
			mi.users--
			dispose := mi.claimDisposal()
			mi.mu.Unlock()
			if dispose {
				mi.dispose()
			}
		})
	}, true
}

func (mi *managedInstance) retire(delay time.Duration) {
	mi.mu.Lock()
	if mi.retired {
		mi.mu.Unlock()
		return
	}
	mi.retired = true
	mi.mu.Unlock()

	ready := func() {
		mi.mu.Lock()
		mi.ready = true
		dispose := mi.claimDisposal()
		mi.mu.Unlock()
		if dispose {
			mi.dispose()
		}
	}
	if _, ok := mi.instance.(InstanceDisposer); ok && delay > 0 {
		time.AfterFunc(delay, ready)
	} else {
		ready()
	}
}

// Called with mu held. Mark disposal before dropping the lock so the timer and
// final request release cannot both dispose the same instance.
func (mi *managedInstance) claimDisposal() bool {
	if !mi.ready || mi.users != 0 || mi.disposed {
		return false
	}
	mi.disposed = true
	return true
}

func (mi *managedInstance) dispose() {
	if disposer, ok := mi.instance.(InstanceDisposer); ok {
		disposer.Dispose()
	}
	activeInstances.Dec()
}
