package backend

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/build/buildinfo"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tracerprovider"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestGetTracingConfig(t *testing.T) {
	for _, tc := range []struct {
		name string

		env             map[string]string
		buildInfoGetter buildinfo.Getter

		expEnabled bool
		expCfg     tracingConfig
	}{
		{
			name:       "disabled",
			env:        nil,
			expEnabled: false,
			expCfg:     tracingConfig{},
		},
		{
			name: "otel with default sampler",
			env: map[string]string{
				PluginTracingOpenTelemetryOTLPAddressEnv:     "127.0.0.1:10000",
				PluginTracingOpenTelemetryOTLPPropagationEnv: "jaeger",
			},
			expEnabled: true,
			expCfg: tracingConfig{
				address:     "127.0.0.1:10000",
				propagation: "jaeger",
				sampler: tracerprovider.SamplerOptions{
					SamplerType: "",
					Param:       1.0, // always sample
					Remote:      tracerprovider.RemoteSamplerOptions{},
				},
			},
		},
		{
			name: "otel with sampler and sampler param",
			env: map[string]string{
				PluginTracingOpenTelemetryOTLPAddressEnv:     "127.0.0.1:10000",
				PluginTracingOpenTelemetryOTLPPropagationEnv: "jaeger",
				PluginTracingSamplerTypeEnv:                  "rateLimiting",
				PluginTracingSamplerParamEnv:                 "0.5",
			},
			expEnabled: true,
			expCfg: tracingConfig{
				address:     "127.0.0.1:10000",
				propagation: "jaeger",
				sampler: tracerprovider.SamplerOptions{
					SamplerType: "rateLimiting",
					Param:       0.5,
					Remote:      tracerprovider.RemoteSamplerOptions{},
				},
			},
		},
		{
			name: "otel with remote sampler",
			env: map[string]string{
				PluginTracingOpenTelemetryOTLPAddressEnv:     "127.0.0.1:10000",
				PluginTracingOpenTelemetryOTLPPropagationEnv: "jaeger",
				PluginTracingSamplerTypeEnv:                  "remote",
				PluginTracingSamplerParamEnv:                 "0.5",
				PluginTracingSamplerRemoteURL:                "127.0.0.1:10001",
			},
			expEnabled: true,
			expCfg: tracingConfig{
				address:     "127.0.0.1:10000",
				propagation: "jaeger",
				sampler: tracerprovider.SamplerOptions{
					SamplerType: "remote",
					Param:       0.5,
					Remote: tracerprovider.RemoteSamplerOptions{
						URL:         "127.0.0.1:10001",
						ServiceName: "grafana-plugin",
					},
				},
			},
		},
		{
			name: "otel with remote sampler and buildinfo service name",
			env: map[string]string{
				PluginTracingOpenTelemetryOTLPAddressEnv:     "127.0.0.1:10000",
				PluginTracingOpenTelemetryOTLPPropagationEnv: "jaeger",
				PluginTracingSamplerTypeEnv:                  "remote",
				PluginTracingSamplerParamEnv:                 "0.5",
				PluginTracingSamplerRemoteURL:                "127.0.0.1:10001",
			},
			buildInfoGetter: buildinfo.GetterFunc(func() (buildinfo.Info, error) {
				return buildinfo.Info{PluginID: "my-example-datasource"}, nil
			}),
			expEnabled: true,
			expCfg: tracingConfig{
				address:     "127.0.0.1:10000",
				propagation: "jaeger",
				sampler: tracerprovider.SamplerOptions{
					SamplerType: "remote",
					Param:       0.5,
					Remote: tracerprovider.RemoteSamplerOptions{
						URL:         "127.0.0.1:10001",
						ServiceName: "my-example-datasource",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for e, v := range tc.env {
				t.Setenv(e, v)
			}
			if tc.buildInfoGetter == nil {
				tc.buildInfoGetter = buildinfo.GetBuildInfo
			}
			cfg := getTracingConfig(tc.buildInfoGetter)
			require.Equal(t, tc.expEnabled, cfg.isEnabled())
			require.Equal(t, tc.expCfg, cfg)

			err := SetupTracer("test", tracing.Opts{})
			require.NoError(t, err)

			tp := otel.GetTracerProvider()
			require.NotNil(t, tp)

			_, ok := tp.(tracerprovider.TracerProvider)
			require.True(t, ok)
		})
	}
}
