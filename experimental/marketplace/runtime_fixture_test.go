package marketplace

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

const runtimeCallTimeout = 30 * time.Second

type fixtureClients struct {
	resource    grpcplugin.ResourceClient
	diagnostics grpcplugin.DiagnosticsClient
	data        grpcplugin.DataClient
}

// startFixture builds the fixture, starts it as a go-plugin subprocess without
// any license available and returns gRPC clients for it.
func startFixture(t *testing.T, packageID string, development bool, args ...string) fixtureClients {
	t.Helper()
	tags := []string{"arrow_json_stdlib"}
	if development {
		tags = append(tags, "marketplace_dev")
	}
	binary := buildFixture(t, fixtureBuildOptions{goos: runtime.GOOS, goarch: runtime.GOARCH, tags: tags, packageID: packageID})

	home, workDir := t.TempDir(), t.TempDir()
	cmd := exec.Command(binary, args...) // #nosec G204 -- binary was built by this test
	cmd.Dir = workDir
	cmd.Env = append(cleanFixtureEnv(),
		"HOME="+home,
		"USERPROFILE="+home,
		"GF_MARKETPLACE_LICENSE_PATH="+filepath.Join(workDir, "missing.jwt"),
		"GF_MARKETPLACE_LICENSE_TEXT=",
	)

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  grpcplugin.ProtocolVersion,
			MagicCookieKey:   grpcplugin.MagicCookieKey,
			MagicCookieValue: grpcplugin.MagicCookieValue,
		},
		Plugins: plugin.PluginSet{
			"diagnostics": &grpcplugin.DiagnosticsGRPCPlugin{},
			"resource":    &grpcplugin.ResourceGRPCPlugin{},
			"data":        &grpcplugin.DataGRPCPlugin{},
		},
		Cmd: cmd,
		// Without this, go-plugin appends the parent environment after cmd.Env and
		// overrides the license settings above.
		SkipHostEnv:      true,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           hclog.New(&hclog.LoggerOptions{Output: io.Discard, Level: hclog.Error}),
		StartTimeout:     runtimeCallTimeout,
	})
	t.Cleanup(client.Kill)

	rpc, err := client.Client()
	require.NoError(t, err)
	dispense := func(name string) any {
		raw, err := rpc.Dispense(name)
		require.NoError(t, err, name)
		return raw
	}
	return fixtureClients{
		resource:    dispense("resource").(grpcplugin.ResourceClient),
		diagnostics: dispense("diagnostics").(grpcplugin.DiagnosticsClient),
		data:        dispense("data").(grpcplugin.DataClient),
	}
}

// cleanFixtureEnv returns the parent environment minus anything licence related.
func cleanFixtureEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		// Upper-case the key: environment variable names are case-insensitive on Windows.
		key, _, _ := strings.Cut(kv, "=")
		key = strings.ToUpper(key)
		if strings.HasPrefix(key, "GF_") || key == "HOME" || key == "USERPROFILE" {
			continue
		}
		env = append(env, kv)
	}
	return env
}

func fixtureContext(appFixture bool) *pluginv2.PluginContext {
	if appFixture {
		return &pluginv2.PluginContext{
			PluginId:            "marketplace-app-fixture",
			AppInstanceSettings: &pluginv2.AppInstanceSettings{LastUpdatedMS: 1},
		}
	}
	return &pluginv2.PluginContext{
		PluginId:                   "marketplace-fixture",
		DataSourceInstanceSettings: &pluginv2.DataSourceInstanceSettings{Name: "fixture", LastUpdatedMS: 1},
	}
}

// callContext returns a timeout context, with the optional "license" gRPC metadata value set.
func callContext(t *testing.T, license []string) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), runtimeCallTimeout)
	t.Cleanup(cancel)
	if len(license) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "license", license[0])
	}
	return ctx
}

func callResource(t *testing.T, c grpcplugin.ResourceClient, license ...string) (code int32, contentType, body string) {
	t.Helper()
	ctx := callContext(t, license)
	stream, err := c.CallResource(ctx, &pluginv2.CallResourceRequest{PluginContext: fixtureContext(true), Path: "test", Method: "GET"})
	require.NoError(t, err)
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return code, contentType, body
		}
		require.NoError(t, err)
		if resp.Code != 0 {
			code = resp.Code
		}
		if h := resp.Headers["Content-Type"]; h != nil && len(h.Values) > 0 {
			contentType = h.Values[0]
		}
		body += string(resp.Body)
	}
}

func checkHealth(t *testing.T, c grpcplugin.DiagnosticsClient, appFixture bool, license ...string) *pluginv2.CheckHealthResponse {
	t.Helper()
	ctx := callContext(t, license)
	resp, err := c.CheckHealth(ctx, &pluginv2.CheckHealthRequest{PluginContext: fixtureContext(appFixture)})
	require.NoError(t, err)
	return resp
}

// TestRuntimeFixtureLicenseStates starts the fixture executables and calls them over gRPC.
// Production builds run without a license and must serve only license errors. Development
// (marketplace_dev) builds stand in for the licensed state: they show that Manage serves
// the plugin's own handlers when the decorator returns the serve options unchanged. No
// signing key is available to mint a valid production token here, so accepting valid and
// grace-period tokens is covered by TestMarketplaceLicenseProductionModeServesLicensedPlugin.
func TestRuntimeFixtureLicenseStates(t *testing.T) {
	if testing.Short() {
		t.Skip("runtime executable fixture")
	}
	t.Parallel()

	t.Run("app production without license", func(t *testing.T) {
		t.Parallel()
		c := startFixture(t, "appfixture", false)
		code, contentType, body := callResource(t, c.resource)
		require.Equal(t, int32(403), code)
		require.Equal(t, "text/plain; charset=utf-8", contentType)
		require.Contains(t, body, "The Marketplace plugin marketplace-app-fixture is not available")
		health := checkHealth(t, c.diagnostics, true)
		require.Equal(t, pluginv2.CheckHealthResponse_ERROR, health.Status)
		require.Equal(t, "Marketplace License Error", health.Message)
	})

	t.Run("app development", func(t *testing.T) {
		t.Parallel()
		c := startFixture(t, "appfixture", true)
		code, _, body := callResource(t, c.resource)
		require.Equal(t, int32(200), code)
		require.Equal(t, "app fixture resource", body)
		health := checkHealth(t, c.diagnostics, true)
		require.Equal(t, pluginv2.CheckHealthResponse_OK, health.Status)
		require.Equal(t, "app fixture healthy", health.Message)
	})

	for _, development := range []bool{false, true} {
		name := "datasource production without license"
		if development {
			name = "datasource development"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := startFixture(t, "marketplacefixture", development)
			ctx, cancel := context.WithTimeout(context.Background(), runtimeCallTimeout)
			defer cancel()
			resp, err := c.data.QueryData(ctx, &pluginv2.QueryDataRequest{
				PluginContext: fixtureContext(false),
				Queries: []*pluginv2.DataQuery{
					{RefId: "A", TimeRange: &pluginv2.TimeRange{}},
					{RefId: "B", TimeRange: &pluginv2.TimeRange{}},
				},
			})
			require.NoError(t, err)
			require.Len(t, resp.Responses, 2)
			for _, refID := range []string{"A", "B"} {
				r := resp.Responses[refID]
				require.NotNil(t, r, refID)
				if development {
					require.Empty(t, r.Error, refID)
				} else {
					require.Equal(t, int32(401), r.Status, refID)
					require.Contains(t, r.Error, "license", refID)
				}
			}
			health := checkHealth(t, c.diagnostics, false)
			if development {
				require.Equal(t, pluginv2.CheckHealthResponse_OK, health.Status)
				require.Equal(t, "datasource fixture healthy", health.Message)
			} else {
				require.Equal(t, pluginv2.CheckHealthResponse_ERROR, health.Status)
				require.Equal(t, "Marketplace License Error", health.Message)
			}
		})
	}
}

// TestRuntimeFixtureQTLVLicenseStates runs the fixtures with -qtlv, which validates the
// "license" gRPC metadata on every request and ignores env and files. No signing key is
// available to mint a valid token here, so acceptance is covered by TestRequestTimeMiddlewareChain.
func TestRuntimeFixtureQTLVLicenseStates(t *testing.T) {
	if testing.Short() {
		t.Skip("runtime executable fixture")
	}
	t.Parallel()

	for name, license := range map[string][]string{
		"no metadata":   nil,
		"unparsable":    {"{"},
		"garbage token": {`{"licenseToken":"garbage","appURL":"http://localhost:3000","validationKeys":""}`},
	} {
		t.Run("app production "+name, func(t *testing.T) {
			t.Parallel()
			c := startFixture(t, "appfixture", false, "-qtlv")
			code, contentType, body := callResource(t, c.resource, license...)
			require.Equal(t, int32(403), code)
			require.Equal(t, "text/plain; charset=utf-8", contentType)
			require.Contains(t, body, "marketplace-app-fixture")
			health := checkHealth(t, c.diagnostics, true, license...)
			require.Equal(t, pluginv2.CheckHealthResponse_ERROR, health.Status)
			require.Equal(t, "Marketplace License Error", health.Message)
		})
	}

	t.Run("app development", func(t *testing.T) {
		t.Parallel()
		c := startFixture(t, "appfixture", true, "-qtlv")
		code, _, body := callResource(t, c.resource)
		require.Equal(t, int32(200), code)
		require.Equal(t, "app fixture resource", body)
		health := checkHealth(t, c.diagnostics, true)
		require.Equal(t, pluginv2.CheckHealthResponse_OK, health.Status)
		require.Equal(t, "app fixture healthy", health.Message)
	})

	for _, development := range []bool{false, true} {
		name := "datasource production"
		if development {
			name = "datasource development"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := startFixture(t, "marketplacefixture", development, "-qtlv")
			resp, err := c.data.QueryData(callContext(t, nil), &pluginv2.QueryDataRequest{
				PluginContext: fixtureContext(false),
				Queries: []*pluginv2.DataQuery{
					{RefId: "A", TimeRange: &pluginv2.TimeRange{}},
					{RefId: "B", TimeRange: &pluginv2.TimeRange{}},
				},
			})
			require.NoError(t, err)
			require.Len(t, resp.Responses, 2)
			for _, refID := range []string{"A", "B"} {
				r := resp.Responses[refID]
				require.NotNil(t, r, refID)
				if development {
					require.Empty(t, r.Error, refID)
				} else {
					require.Equal(t, int32(401), r.Status, refID)
				}
			}
			// The middleware must not block metrics collection.
			_, err = c.diagnostics.CollectMetrics(callContext(t, nil), &pluginv2.CollectMetricsRequest{PluginContext: fixtureContext(false)})
			require.NoError(t, err)
		})
	}
}
