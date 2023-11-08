package backend

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/build"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tracerprovider"
)

const (
	// PluginProfilerEnvDeprecated is a deprecated constant for the GF_PLUGINS_PROFILER environment variable used to enable pprof.
	PluginProfilerEnvDeprecated = "GF_PLUGINS_PROFILER"
	// PluginProfilingEnabledEnv is a constant for the GF_PLUGIN_PROFILING_ENABLED environment variable used to enable pprof.
	PluginProfilingEnabledEnv = "GF_PLUGIN_PROFILING_ENABLED"

	// PluginProfilerPortEnvDeprecated is a constant for the GF_PLUGINS_PROFILER_PORT environment variable use to specify a pprof port (default 6060).
	PluginProfilerPortEnvDeprecated = "GF_PLUGINS_PROFILER_PORT" // nolint:gosec
	// PluginProfilingPortEnv is a constant for the GF_PLUGIN_PROFILING_PORT environment variable use to specify a pprof port (default 6060).
	PluginProfilingPortEnv = "GF_PLUGIN_PROFILING_PORT" // nolint:gosec

	// PluginTracingOpenTelemetryOTLPAddressEnv is a constant for the GF_INSTANCE_OTLP_ADDRESS
	// environment variable used to specify the OTLP address.
	PluginTracingOpenTelemetryOTLPAddressEnv = "GF_INSTANCE_OTLP_ADDRESS" // nolint:gosec
	// PluginTracingOpenTelemetryOTLPPropagationEnv is a constant for the GF_INSTANCE_OTLP_PROPAGATION
	// environment variable used to specify the OTLP propagation format.
	PluginTracingOpenTelemetryOTLPPropagationEnv = "GF_INSTANCE_OTLP_PROPAGATION"

	PluginTracingSamplerTypeEnv   = "GF_INSTANCE_OTLP_SAMPLER_TYPE"
	PluginTracingSamplerParamEnv  = "GF_INSTANCE_OTLP_SAMPLER_PARAM"
	PluginTracingSamplerRemoteURL = "GF_INSTANCE_OTLP_SAMPLER_REMOTE_URL"

	// PluginVersionEnv is a constant for the GF_PLUGIN_VERSION environment variable containing the plugin's version.
	// Deprecated: Use build.GetBuildInfo().Version instead.
	PluginVersionEnv = "GF_PLUGIN_VERSION"

	defaultServiceName = "grafana-plugin"
)

// SetupPluginEnvironment will read the environment variables and apply the
// standard environment behavior.
//
// As the SDK evolves, this will likely change.
//
// Currently, this function enables and configures profiling with pprof.
func SetupPluginEnvironment(pluginID string) {
	setupProfiler(pluginID)
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

	Logger.Debug("Profiler", "enabled", profilerEnabled)
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

func getTracerCustomAttributes(pluginID string) []attribute.KeyValue {
	var customAttributes []attribute.KeyValue
	// Add plugin id and version to custom attributes
	// Try to get plugin version from build info
	// If not available, fallback to environment variable
	var pluginVersion string
	buildInfo, err := build.GetBuildInfo()
	if err != nil {
		Logger.Debug("Failed to get build info", "error", err)
	} else {
		pluginVersion = buildInfo.Version
	}
	if pluginVersion == "" {
		if pv, ok := os.LookupEnv(PluginVersionEnv); ok {
			pluginVersion = pv
		}
	}
	customAttributes = []attribute.KeyValue{
		semconv.ServiceNameKey.String(pluginID),
		semconv.ServiceVersionKey.String(pluginVersion),
	}
	return customAttributes
}

// SetupTracer sets up the global OTEL trace provider and tracer.
func SetupTracer(pluginID string, tracingOpts tracing.Opts) error {
	// Set up tracing
	tracingCfg := getTracingConfig(build.GetBuildInfo)
	if tracingCfg.IsEnabled() {
		// Append custom attributes to the default ones
		tracingOpts.CustomAttributes = append(getTracerCustomAttributes(pluginID), tracingOpts.CustomAttributes...)

		// Initialize global tracer provider
		tp, err := tracerprovider.NewTracerProvider(tracingCfg.address, tracingCfg.sampler, tracingOpts)
		if err != nil {
			return fmt.Errorf("new trace provider: %w", err)
		}
		pf, err := tracerprovider.NewTextMapPropagator(tracingCfg.propagation)
		if err != nil {
			return fmt.Errorf("new propagator format: %w", err)
		}
		tracerprovider.InitGlobalTracerProvider(tp, pf)

		// Initialize global tracer for plugin developer usage
		tracing.InitDefaultTracer(otel.Tracer(pluginID))
	}

	Logger.Debug(
		"Tracing",
		"enabled", tracingCfg.IsEnabled(),
		"propagation", tracingCfg.propagation,
		"samplerType", tracingCfg.sampler.SamplerType,
		"samplerParam", tracingCfg.sampler.Param,
		"samplerRemote", tracingCfg.sampler.Remote,
	)
	return nil
}

// tracingConfig contains the configuration for OTEL tracing.
type tracingConfig struct {
	address     string
	propagation string

	sampler tracerprovider.SamplerOptions
}

// IsEnabled returns true if OTEL tracing is enabled.
func (c tracingConfig) IsEnabled() bool {
	return c.address != ""
}

// getTracingConfig returns a new tracingConfig based on the current environment variables.
func getTracingConfig(buildInfoGetter build.InfoGetter) tracingConfig {
	var otelAddr, otelPropagation, samplerRemoteURL, samplerParamString string
	var samplerType tracerprovider.SamplerType
	var samplerParam float64
	otelAddr, ok := os.LookupEnv(PluginTracingOpenTelemetryOTLPAddressEnv)
	if ok {
		// Additional OTEL config
		otelPropagation = os.Getenv(PluginTracingOpenTelemetryOTLPPropagationEnv)

		// Sampling config
		samplerType = tracerprovider.SamplerType(os.Getenv(PluginTracingSamplerTypeEnv))
		samplerRemoteURL = os.Getenv(PluginTracingSamplerRemoteURL)
		samplerParamString = os.Getenv(PluginTracingSamplerParamEnv)
		var err error
		samplerParam, err = strconv.ParseFloat(samplerParamString, 64)
		if err != nil {
			// Default value if invalid float is provided is 1.0 (AlwaysSample)
			samplerParam = 1.0
		}
	}

	var serviceName string
	if samplerType == tracerprovider.SamplerTypeRemote {
		// Use plugin id as service name, if possible. Otherwise, use a generic default value.
		bi, _ := buildInfoGetter.GetInfo()
		serviceName = bi.PluginID
		if serviceName == "" {
			serviceName = defaultServiceName
		}
	}

	return tracingConfig{
		address:     otelAddr,
		propagation: otelPropagation,
		sampler: tracerprovider.SamplerOptions{
			SamplerType: samplerType,
			Param:       samplerParam,
			Remote: tracerprovider.RemoteSamplerOptions{
				URL:         samplerRemoteURL,
				ServiceName: serviceName,
			},
		},
	}
}
