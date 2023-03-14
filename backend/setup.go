package backend

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
)

var (
	// PluginProfilerEnvDeprecated is a deprecated constant for the GF_PLUGINS_PROFILER environment variable used to enable pprof.
	PluginProfilerEnvDeprecated = "GF_PLUGINS_PROFILER"
	// PluginProfilingEnabledEnv is a constant for the GF_PLUGIN_PROFILING_ENABLED environment variable used to enable pprof.
	PluginProfilingEnabledEnv = "GF_PLUGIN_PROFILING_ENABLED"

	// PluginProfilerPortEnvDeprecated is a constant for the GF_PLUGINS_PROFILER_PORT environment variable use to specify a pprof port (default 6060).
	PluginProfilerPortEnvDeprecated = "GF_PLUGINS_PROFILER_PORT"
	// PluginProfilingPortEnv is a constant for the GF_PLUGIN_PROFILING_PORT environment variable use to specify a pprof port (default 6060).
	PluginProfilingPortEnv = "GF_PLUGIN_PROFILING_PORT"

	// PluginTracingOpenTelemetryOTLPAddressEnv is a constant for the GF_TRACING_OPENTELEMETRY_OTLP_ADDRESS
	// environment variable used to specify the OTLP Address.
	PluginTracingOpenTelemetryOTLPAddressEnv = "GF_TRACING_OPENTELEMETRY_OTLP_ADDRESS"
	// PluginTracingOpenTelemetryOTLPPropagationEnv is a constant for the GF_TRACING_OPENTELEMETRY_OTLP_PROPAGATION
	// environment variable used to specify the OTLP propagation format.
	PluginTracingOpenTelemetryOTLPPropagationEnv = "GF_TRACING_OPENTELEMETRY_OTLP_PROPAGATION"
)

// SetupPluginEnvironment will read the environment variables and apply the
// standard environment behavior.
//
// As the SDK evolves, this will likely change.
//
// Currently this function enables and configures profiling with pprof and the global tracer.
func SetupPluginEnvironment(pluginID string, tracingOpts tracing.Opts) error {
	setupProfiler(pluginID)
	if err := setupTracer(pluginID, tracingOpts); err != nil {
		return fmt.Errorf("setup tracer: %w", err)
	}
	return nil
}

func setupProfiler(pluginID string) {
	// Enable profiler
	profilerEnabled := false
	if value, ok := os.LookupEnv(PluginProfilerEnvDeprecated); ok {
		// compare value to plugin name
		if value == pluginID {
			profilerEnabled = true
		}
	} else if value, ok = os.LookupEnv(PluginProfilingEnabledEnv); ok {
		if value == "true" {
			profilerEnabled = true
		}
	}

	Logger.Info("Profiler", "enabled", profilerEnabled)
	if profilerEnabled {
		profilerPort := "6060"
		for _, env := range []string{PluginProfilerPortEnvDeprecated, PluginProfilingPortEnv} {
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
			//nolint:gosec
			if err := http.ListenAndServe(portConfig, r); err != nil {
				Logger.Error("Error Running profiler", "error", err)
			}
		}()
	}
}

func setupTracer(pluginID string, tracingOpts tracing.Opts) error {
	// Set up tracing
	tracingCfg := getTracingConfig()
	if tracingCfg.IsEnabled() {
		tp, err := tracing.NewTraceProvider(tracingCfg.Address, pluginID, tracingOpts)
		if err != nil {
			return fmt.Errorf("new trace provider: %w", err)
		}
		tracing.InitGlobalTraceProvider(tp, tracing.NewPropagatorFormat(tracingCfg.Propagation))
	}
	Logger.Info("Tracing", "enabled", tracingCfg.IsEnabled(), "propagation", tracingCfg.Propagation)
	return nil
}

type tracingConfig struct {
	Address     string
	Propagation tracing.PropagatorFormat
}

func (c tracingConfig) IsEnabled() bool {
	return c.Address != ""
}

func getTracingConfig() tracingConfig {
	var otelAddr, otelPropagation string
	otelAddr, ok := os.LookupEnv(PluginTracingOpenTelemetryOTLPAddressEnv)
	if ok {
		otelPropagation = os.Getenv(PluginTracingOpenTelemetryOTLPPropagationEnv)
	}
	return tracingConfig{
		Address:     otelAddr,
		Propagation: tracing.PropagatorFormat(otelPropagation),
	}
}
