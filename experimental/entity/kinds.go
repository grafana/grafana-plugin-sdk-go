package entity

import (
	"fmt"
	sync "sync"
)

type Kinds struct {
	kinds  map[string]Kind
	lock   sync.RWMutex
	suffix suffixMap
}

type suffixMap struct {
	kind  Kind
	kinds map[string]suffixMap
}

func NewKindRegistry(k ...Kind) (*Kinds, error) {
	kinds := &Kinds{
		kinds:  make(map[string]Kind),
		suffix: suffixMap{},
	}
	err := kinds.Register(k...)
	if err != nil {
		return nil, err
	}
	return kinds, nil
}

func (r *Kinds) Register(kinds ...Kind) error {
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
		if info.FileSuffix == "" {
			return fmt.Errorf("kind must have a suffix")
		}
		// TODO... check if the suffix is registered

		r.kinds[info.ID] = k
	}
	return nil
}

func (r *Kinds) Get(id string) Kind {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.kinds[id]
}

func (r *Kinds) List() []Kind {
	r.lock.RLock()
	defer r.lock.RUnlock()

	kinds := make([]Kind, 0, len(r.kinds))
	for _, k := range r.kinds {
		kinds = append(kinds, k)
	}
	return kinds
}

func (r *Kinds) GetBySuffix(path string) Kind {
	// TODO!!!
	return nil
}

// 	Register(k ... Kind) error
// 	GetKind(k string) Kind
// 	List() []Kind
// 	GetFromSuffix(path string) Kind
// }

// var kinds = []EntityKindInfo{
// 	{
// 		ID:         "dashboard",
// 		FileSuffix: "-dash.json",
// 	},
// 	{
// 		ID:         "alert",
// 		FileSuffix: "-alert.json",
// 	},
// 	{
// 		ID:         "datasource",
// 		FileSuffix: "-ds.json",
// 	},
// 	{
// 		ID:         "playlist",
// 		FileSuffix: "-playlist.json",
// 	},
// 	{
// 		ID:          "annotation",
// 		Description: "Single annotation event",
// 		FileSuffix:  "-anno.json",
// 	},
// 	// ???
// 	{
// 		ID:         "readme",
// 		FileSuffix: "README.md",
// 	},
// 	{
// 		ID:         "folder",
// 		FileSuffix: "__folder.json",
// 	},
// 	// Data
// 	{
// 		ID:         "dataFrame",
// 		FileSuffix: "-df.json",
// 		Category:   "Data",
// 	},
// 	{
// 		ID:          "dataQueryResponse",
// 		Description: "query result format",
// 		FileSuffix:  "-dqr.json",
// 		Category:    "Data",
// 	},
// 	{
// 		ID:         "CSV",
// 		FileSuffix: ".csv",
// 		Category:   "Data",
// 	},
// 	{
// 		ID:         "GeoJSON",
// 		FileSuffix: ".geojson",
// 		Category:   "Data",
// 	},
// 	{
// 		ID:         "WorldMap location lookup",
// 		FileSuffix: "-wm.json",
// 		Category:   "Data",
// 	},
// 	// Images (binary)
// 	{
// 		ID:         "SVG",
// 		FileSuffix: ".svg",
// 		Category:   "Image",
// 	},
// 	{
// 		ID:         "PNG",
// 		FileSuffix: ".png",
// 		Category:   "Image",
// 	},
// 	{
// 		ID:         "JPEG",
// 		FileSuffix: ".jpg",
// 		Category:   "Image",
// 	},
// 	{
// 		ID:         "GIF",
// 		FileSuffix: ".gif",
// 		Category:   "Image",
// 	},
// }

// func GetXXX() {
// 	for _, k := range kinds {
// 		fmt.Printf("%+v\n", k)
// 	}
// }
