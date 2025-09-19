// Package buildinfo provides build information functionality for Grafana plugins.
// It allows plugins to access build-time information such as version, plugin ID, and build time.
package buildinfo

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
)

// Command line flags for build info mode
var (
	buildInfoMode = flag.Bool("buildinfo", false, "print build info and exit")
	versionMode   = flag.Bool("version", false, "print version and exit")
)

// buildInfoJSON is set from -X linker flag during compilation
var buildInfoJSON string

// Info represents build information for a plugin
type Info struct {
	Time     int64  `json:"time,omitempty"`
	PluginID string `json:"pluginID,omitempty"`
	Version  string `json:"version,omitempty"`
}

// InfoGetter is an interface with a method for returning the build info.
type InfoGetter interface {
	// GetInfo returns the build info.
	GetInfo() (Info, error)
}

// InfoGetterFunc can be used to adapt ordinary functions into types satisfying the InfoGetter interface.
type InfoGetterFunc func() (Info, error)

// GetInfo implements the InfoGetter interface.
func (f InfoGetterFunc) GetInfo() (Info, error) {
	return f()
}

// GetBuildInfo is the default InfoGetter that returns the build information that was compiled into the binary using:
// -X `github.com/grafana/grafana-plugin-sdk-go/internal/buildinfo.buildInfoJSON={...}`
var GetBuildInfo = InfoGetterFunc(func() (Info, error) {
	v := Info{}
	if buildInfoJSON == "" {
		return v, fmt.Errorf("build info was not set when this was compiled")
	}
	err := json.Unmarshal([]byte(buildInfoJSON), &v)
	return v, err
})

// InfoModeEnabled returns true if the plugin should run in build info mode
// (-buildinfo or -version flags provided).
func InfoModeEnabled() bool {
	flag.Parse()
	return *buildInfoMode || *versionMode
}

// RunInfoMode runs the plugin in build info mode, which prints the build info (or just the version) to stdout and returns.
// The caller should call os.Exit right after.
func RunInfoMode() error {
	if !InfoModeEnabled() {
		return errors.New("build info mode not enabled")
	}
	bi, err := GetBuildInfo()
	if err != nil {
		return fmt.Errorf("get build info: %w", err)
	}
	bib, err := json.Marshal(bi)
	if err != nil {
		return fmt.Errorf("marshal build info: %w", err)
	}
	if *versionMode {
		fmt.Println(bi.Version)
	} else {
		fmt.Println(string(bib))
	}
	return nil
}
