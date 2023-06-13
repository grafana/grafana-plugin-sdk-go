package tenanttest

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/grafana/grafana-plugin-sdk-go/internal/automanagement"
	"github.com/grafana/grafana-plugin-sdk-go/internal/standalone"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tenant"
)

const (
	tenantID1 = "abc123"
	tenantID2 = "def456"
	addr      = "127.0.0.1:8000"
)

// A test to verify the impact tenant ID (passed via context) has on plugin instance management
func TestTenantWithPluginInstanceManagement(t *testing.T) {
	factoryInvocations := 0
	factory := datasource.InstanceFactoryFunc(func(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		factoryInvocations++
		return &testPlugin{
			settings: dataSourceSettings{},
		}, nil
	})
	instancePrvdr := datasource.NewInstanceProvider(factory)
	instanceMgr := instancemgmt.New(instancePrvdr)
	handler := automanagement.NewManager(instanceMgr)

	pCtx := backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{}}
	qdr := &backend.QueryDataRequest{PluginContext: pCtx}
	crr := &backend.CallResourceRequest{PluginContext: pCtx}
	chr := &backend.CheckHealthRequest{PluginContext: pCtx}
	responseSender := newTestCallResourceResponseSender()

	go func() {
		err := backend.GracefulStandaloneServe(backend.ServeOpts{
			QueryDataHandler:    handler,
			CallResourceHandler: handler,
			StreamHandler:       handler,
			CheckHealthHandler:  handler,
		}, standalone.NewServerSettings(addr))
		require.NoError(t, err)
	}()

	pc, shutdown, err := newPluginClient(addr)
	require.NoError(t, err)
	defer func() {
		err = shutdown()
		require.NoError(t, err)
	}()

	t.Run("Request without tenant information creates an instance", func(t *testing.T) {
		ctx := context.Background()
		resp, err := pc.QueryData(ctx, qdr)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, factoryInvocations)

		err = pc.CallResource(ctx, crr, responseSender)
		require.NoError(t, err)
		require.Equal(t, 1, factoryInvocations)

		t.Run("Request from tenant #1 creates new instance", func(t *testing.T) {
			ctx = metadata.AppendToOutgoingContext(context.Background(), tenant.CtxKey, tenantID1)
			resp, err = pc.QueryData(ctx, qdr)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, 2, factoryInvocations)

			// subsequent requests from tenantID1 with same settings will reuse instance
			resp, err = pc.QueryData(ctx, qdr)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, 2, factoryInvocations)

			var chRes *backend.CheckHealthResult
			chRes, err = pc.CheckHealth(ctx, chr)
			require.NoError(t, err)
			require.NotNil(t, chRes)
			require.Equal(t, 2, factoryInvocations)

			t.Run("Request from tenant #2 creates new instance", func(t *testing.T) {
				ctx = metadata.AppendToOutgoingContext(context.Background(), tenant.CtxKey, tenantID2)
				resp, err = pc.QueryData(ctx, qdr)
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, 3, factoryInvocations)

				// subsequent requests from tenantID2 with same settings will reuse instance
				err = pc.CallResource(ctx, crr, responseSender)
				require.NoError(t, err)
				require.Equal(t, 3, factoryInvocations)
			})

			// subsequent requests from tenantID1 with same settings will reuse instance
			ctx = metadata.AppendToOutgoingContext(context.Background(), tenant.CtxKey, tenantID1)
			resp, err = pc.QueryData(ctx, qdr)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, 3, factoryInvocations)

			chRes, err = pc.CheckHealth(ctx, chr)
			require.NoError(t, err)
			require.NotNil(t, chRes)
			require.Equal(t, 3, factoryInvocations)
		})
	})
}

type testPlugin struct {
	settings dataSourceSettings
}

type dataSourceSettings struct{}

func (p *testPlugin) QueryData(_ context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return backend.NewQueryDataResponse(), nil
}

func (p *testPlugin) CallResource(_ context.Context, _ *backend.CallResourceRequest, _ backend.CallResourceResponseSender) error {
	return nil
}

func (p *testPlugin) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{}, nil
}

type testPluginClient struct {
	dataClient        pluginv2.DataClient
	diagnosticsClient pluginv2.DiagnosticsClient
	resourceClient    pluginv2.ResourceClient
}

type shutdownFunc func() error

var noShutdown = shutdownFunc(func() error {
	return nil
})

func newPluginClient(addr string) (*testPluginClient, shutdownFunc, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, noShutdown, err
	}

	plugin := &testPluginClient{
		diagnosticsClient: pluginv2.NewDiagnosticsClient(conn),
		dataClient:        pluginv2.NewDataClient(conn),
		resourceClient:    pluginv2.NewResourceClient(conn),
	}

	return plugin, func() error {
		return conn.Close()
	}, nil
}

func (p *testPluginClient) CheckHealth(ctx context.Context, r *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	req := &pluginv2.CheckHealthRequest{
		PluginContext: backend.ToProto().PluginContext(r.PluginContext),
	}

	resp, err := p.diagnosticsClient.CheckHealth(ctx, req)
	if err != nil {
		return nil, err
	}

	return backend.FromProto().CheckHealthResponse(resp), nil
}

func (p *testPluginClient) CallResource(ctx context.Context, r *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	protoReq := backend.ToProto().CallResourceRequest(r)
	protoStream, err := p.resourceClient.CallResource(ctx, protoReq)
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			return errors.New("method not implemented")
		}

		return err
	}

	for {
		protoResp, err := protoStream.Recv()
		if err != nil {
			if status.Code(err) == codes.Unimplemented {
				return errors.New("method not implemented")
			}

			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if err = sender.Send(backend.FromProto().CallResourceResponse(protoResp)); err != nil {
			return err
		}
	}
}

func (p *testPluginClient) QueryData(ctx context.Context, r *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	req := backend.ToProto().QueryDataRequest(r)

	resp, err := p.dataClient.QueryData(ctx, req)
	if err != nil {
		return nil, err
	}

	return backend.FromProto().QueryDataResponse(resp)
}

type testCallResourceResponseSender struct{}

func newTestCallResourceResponseSender() *testCallResourceResponseSender {
	return &testCallResourceResponseSender{}
}

func (s *testCallResourceResponseSender) Send(resp *backend.CallResourceResponse) error {
	return nil
}
