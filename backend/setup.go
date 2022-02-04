package backend

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
)

var (
	// PluginProfilerEnvDeprecated is a deprecated constant for the GF_PLUGINS_PROFILER environment variable used to enable pprof.
	PluginProfilerEnvDeprecated = "GF_PLUGINS_PROFILER"
	// PluginProfilerEnv is a constant for the GF_PLUGIN_PROFILER environment variable used to enable pprof.
	PluginProfilerEnv = "GF_PLUGIN_PROFILER"

	// PluginProfilerPortEnvDeprecated is a constant for the GF_PLUGINS_PROFILER_PORT environment variable use to specify a pprof port (default 6060).
	PluginProfilerPortEnvDeprecated = "GF_PLUGINS_PROFILER_PORT"
	// PluginProfilerPortEnv is a constant for the GF_PLUGIN_PROFILER_PORT environment variable use to specify a pprof port (default 6060).
	PluginProfilerPortEnv = "GF_PLUGIN_PROFILER_PORT"
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
	if value, ok := os.LookupEnv(PluginProfilerEnvDeprecated); ok {
		// compare value to plugin name
		if value == pluginID {
			profilerEnabled = true
		}
	} else if value, ok = os.LookupEnv(PluginProfilerEnv); ok {
		if value == "true" {
			profilerEnabled = true
		}
	}

	Logger.Info("Profiler", "enabled", profilerEnabled)
	if profilerEnabled {
		profilerPort := "6060"
		for _, env := range []string{PluginProfilerPortEnvDeprecated, PluginProfilerPortEnv} {
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
