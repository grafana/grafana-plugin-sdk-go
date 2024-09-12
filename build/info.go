package build

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// set from -X
var buildInfoJSON string

// exposed for testing.
var now = time.Now

// Info See also PluginBuildInfo in https://github.com/grafana/grafana/blob/master/pkg/plugins/models.go
type Info struct {
	Time     int64  `json:"time,omitempty"`
	PluginID string `json:"pluginID,omitempty"`
	Version  string `json:"version,omitempty"`
}

// this will append build flags -- the keys are picked to match existing
// grafana build flags from bra
func (v Info) appendFlags(flags map[string]string) {
	if v.PluginID != "" {
		flags["main.pluginID"] = v.PluginID
	}
	if v.Version != "" {
		flags["main.version"] = v.Version
	}

	out, err := json.Marshal(v)
	if err == nil {
		flags["github.com/grafana/grafana-plugin-sdk-go/build.buildInfoJSON"] = string(out)
	}
}

func getEnvironment(check ...string) string {
	for _, key := range check {
		if strings.HasPrefix(key, "> ") {
			parts := strings.Split(key, " ")
			cmd := exec.Command(parts[1], parts[2:]...) // #nosec G204
			out, err := cmd.CombinedOutput()
			if err == nil && len(out) > 0 {
				str := strings.TrimSpace(string(out))
				if strings.Index(str, " ") > 0 {
					continue // skip any output that has spaces
				}
				return str
			}
			continue
		}

		val := os.Getenv(key)
		if val != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

// InfoGetter is an interface with a method for returning the build info.
type InfoGetter interface {
	// GetInfo returns the build info.
	GetInfo() (Info, error)
}

// InfoGetterFunc can be used to adapt ordinary functions into types satisfying the InfoGetter interface .
type InfoGetterFunc func() (Info, error)

func (f InfoGetterFunc) GetInfo() (Info, error) {
	return f()
}

// GetBuildInfo is the default InfoGetter that returns the build information that was compiled into the binary using:
// -X `github.com/grafana/grafana-plugin-sdk-go/build.buildInfoJSON={...}`
var GetBuildInfo = InfoGetterFunc(func() (Info, error) {
	v := Info{}
	if buildInfoJSON == "" {
		return v, fmt.Errorf("build info was now set when this was compiled")
	}
	err := json.Unmarshal([]byte(buildInfoJSON), &v)
	return v, err
})
