package backend

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
)

var (
	// PluginProfilerEnvs is a list of valid environment variables used to enable pprof.
	PluginProfilerEnvs = []string{"GF_PLUGINS_PROFILER", "GF_PLUGIN_PROFILER"}

	// PluginProfilerPortEnvs is a list of valid environment variable used to specify a pprof port (default 6060).
	PluginProfilerPortEnvs = []string{"GF_PLUGINS_PROFILER_PORT", "GF_PLUGIN_PROFILER_PORT"}
)

// SetupPluginEnvironment will read the environment variables and apply the
// standard environment behavior.
//
// As the SDK evolves, this will likely change.
//
// Currently this function enables and configures profiling with pprof.
func SetupPluginEnvironment(pluginID string) {
	// Enable profiler
	profilerEnabled := false
	for _, env := range PluginProfilerEnvs {
		if value, ok := os.LookupEnv(env); ok {
			// compare value to plugin name
			if value == pluginID {
				profilerEnabled = true
			}
			break
		}
	}
	Logger.Info("Profiler", "enabled", profilerEnabled)
	if profilerEnabled {
		profilerPort := "6060"
		for _, env := range PluginProfilerPortEnvs {
			if value, ok := os.LookupEnv(env); ok {
				profilerPort = value
				break
			}
		}

		Logger.Info("Profiler", "port", profilerPort)
		portConfig := fmt.Sprintf(":%s", profilerPort)

		r := http.NewServeMux()
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)

		go func() {
			if err := http.ListenAndServe(portConfig, r); err != nil {
				Logger.Error("Error Running profiler: %s", err.Error())
			}
		}()
	}
}
