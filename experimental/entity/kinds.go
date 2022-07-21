package entity

import (
	"fmt"
	"strings"
	sync "sync"
)

type kinds struct {
	kinds  map[string]Kind
	lock   sync.RWMutex
	suffix suffixMap
}

var _ KindRegistry = &kinds{}

func NewKindRegistry(k ...Kind) (KindRegistry, error) {
	kinds := &kinds{
		kinds:  make(map[string]Kind),
		suffix: suffixMap{},
	}
	err := kinds.Register(k...)
	if err != nil {
		return nil, err
	}
	return kinds, nil
}

// Register adds additional kinds to the registry.
// This will throw an error if duplicate IDs or file extensions exist
func (r *kinds) Register(kinds ...Kind) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, k := range kinds {
		info := k.Info()
		if info.ID == "" {
			return fmt.Errorf("kind must have a name")
		}
		if r.kinds[info.ID] != nil {
			return fmt.Errorf("kind already registered: %s", info.ID)
		}
		if info.PathSuffix == "" {
			return fmt.Errorf("kind must have a suffix")
		}
		if strings.ContainsAny(info.PathSuffix, "$%*();#@/\\") {
			return fmt.Errorf("invalid suffix")
		}

		size := len(info.PathSuffix)
		runes := make([]byte, size) // each characeter
		for i := size - 1; i >= 0; i-- {
			runes[i] = info.PathSuffix[i]
		}

		err := r.suffix.register(k, runes)
		if err != nil {
			return err
		}
		r.kinds[info.ID] = k
	}
	return nil
}

// Get looks up a Kind from ID
func (r *kinds) Get(id string) Kind {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.kinds[id]
}

// List shows all supported kinds
func (r *kinds) List() []Kind {
	r.lock.RLock()
	defer r.lock.RUnlock()

	kinds := make([]Kind, 0, len(r.kinds))
	for _, k := range r.kinds {
		kinds = append(kinds, k)
	}
	return kinds
}

// GetBySuffix finds the kind registered to the file extension
func (r *kinds) GetFromSuffix(path string) Kind {
	return r.suffix.find(path, len(path)-1, nil)
}

// Reverse order lookup
type suffixMap struct {
	// The selected kind
	kind Kind

	// non-null when more suffixes may match
	kinds map[byte]*suffixMap
}

func (s *suffixMap) find(path string, idx int, match Kind) Kind {
	if idx >= 0 {
		k := path[idx]
		sub, ok := s.kinds[k]
		if ok {
			if sub.kind != nil {
				match = sub.kind
			}
			return sub.find(path, idx-1, match)
		}
	}
	return match
}

func (s *suffixMap) register(k Kind, runes []byte) error {
	if s.kinds == nil {
		s.kinds = make(map[byte]*suffixMap)
	}

	count := len(runes)
	if count < 1 {
		return fmt.Errorf("invalid state")
	}
	if count == 1 {
		prev, ok := s.kinds[runes[0]]
		if ok {
			if prev.kind != nil {
				return fmt.Errorf("suffix already registered for: %s", k.Info().PathSuffix)
			}
			prev.kind = k
		} else {
			s.kinds[runes[0]] = &suffixMap{kind: k}
		}
		return nil
	}

	last := runes[count-1]
	rest := runes[0 : count-1]

	prev, ok := s.kinds[last]
	if !ok {
		prev = &suffixMap{}
		s.kinds[last] = prev
	}
	return prev.register(k, rest)
}
