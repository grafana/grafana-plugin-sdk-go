package build

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/build/info"
)

var now = time.Now // allow override for testing

// Deprecated: Use github.com/grafana/grafana-plugin-sdk-go/build/info.Info instead.
type Info = info.Info

// Deprecated: Use github.com/grafana/grafana-plugin-sdk-go/build/info.Getter instead.
type InfoGetter = info.Getter

// Deprecated: Use github.com/grafana/grafana-plugin-sdk-go/build/info.GetterFunc instead.
type InfoGetterFunc = info.GetterFunc

// Deprecated: Use github.com/grafana/grafana-plugin-sdk-go/build/info.GetBuildInfo instead.
var GetBuildInfo = info.GetBuildInfo
