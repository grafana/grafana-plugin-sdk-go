package backend

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"

	hclog "github.com/hashicorp/go-hclog"
)

func registerPProfHandlers(r *http.ServeMux) {
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// PProfSetup sets up the plugin to host PPRof debug info
func PProfSetup(pluginID string, pluginLogger hclog.Logger) {
	// check if pprof should be started
	// GF_PLUGINS_PROFILER=pluginname
	profilerEnabled := false
	if value, ok := os.LookupEnv("GF_PLUGINS_PROFILER"); ok {
		// compare value to plugin name
		if value == pluginID {
			profilerEnabled = true
		}
	} else {
		pluginLogger.Info("Profiler using default setting: false")
	}
	pluginLogger.Info("Profiler", "enabled", profilerEnabled)

	profilerPort := "6060"
	if value, ok := os.LookupEnv("GF_PLUGINS_PROFILER_PORT"); ok {
		profilerPort = value
	}

	if profilerEnabled {
		pluginLogger.Info("Profiler", "port", profilerPort)
		portConfig := fmt.Sprintf(":%s", profilerPort)
		m := http.NewServeMux()
		registerPProfHandlers(m)
		go func() {
			if err := http.ListenAndServe(portConfig, m); err != nil {
				log.Fatal(err)
			}
		}()
	}
}
