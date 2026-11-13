//go:build !marketplace_dev

package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// recordingHandler is a final handler that records which endpoints reached it.
type recordingHandler struct {
	backend.BaseHandler
	calls []string
}

func (r *recordingHandler) QueryData(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	r.calls = append(r.calls, "QueryData")
	return backend.NewQueryDataResponse(), nil
}

func (r *recordingHandler) QueryChunkedData(context.Context, *backend.QueryChunkedDataRequest, backend.ChunkedDataWriter) error {
	r.calls = append(r.calls, "QueryChunkedData")
	return nil
}

func (r *recordingHandler) CallResource(context.Context, *backend.CallResourceRequest, backend.CallResourceResponseSender) error {
	r.calls = append(r.calls, "CallResource")
	return nil
}

func (r *recordingHandler) CheckHealth(context.Context, *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	r.calls = append(r.calls, "CheckHealth")
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk}, nil
}

func (r *recordingHandler) CollectMetrics(context.Context, *backend.CollectMetricsRequest) (*backend.CollectMetricsResult, error) {
	r.calls = append(r.calls, "CollectMetrics")
	return &backend.CollectMetricsResult{}, nil
}

func (r *recordingHandler) SubscribeStream(context.Context, *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	r.calls = append(r.calls, "SubscribeStream")
	return &backend.SubscribeStreamResponse{Status: backend.SubscribeStreamStatusOK}, nil
}

func (r *recordingHandler) PublishStream(context.Context, *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	r.calls = append(r.calls, "PublishStream")
	return &backend.PublishStreamResponse{Status: backend.PublishStreamStatusOK}, nil
}

func (r *recordingHandler) RunStream(context.Context, *backend.RunStreamRequest, *backend.StreamSender) error {
	r.calls = append(r.calls, "RunStream")
	return nil
}

func (r *recordingHandler) ValidateAdmission(context.Context, *backend.AdmissionRequest) (*backend.ValidationResponse, error) {
	r.calls = append(r.calls, "ValidateAdmission")
	return &backend.ValidationResponse{Allowed: true}, nil
}

func (r *recordingHandler) MutateAdmission(context.Context, *backend.AdmissionRequest) (*backend.MutationResponse, error) {
	r.calls = append(r.calls, "MutateAdmission")
	return &backend.MutationResponse{Allowed: true}, nil
}

func (r *recordingHandler) ConvertObjects(context.Context, *backend.ConversionRequest) (*backend.ConversionResponse, error) {
	r.calls = append(r.calls, "ConvertObjects")
	return &backend.ConversionResponse{}, nil
}

// licensedCalls invoke every endpoint the license middleware checks, ignoring the results.
// The refusal responses are covered by the invalidLicenseHandler tests.
var licensedCalls = []struct {
	name   string
	invoke func(ctx context.Context, h backend.Handler)
}{
	{"QueryData", func(ctx context.Context, h backend.Handler) { _, _ = h.QueryData(ctx, &backend.QueryDataRequest{}) }},
	{"QueryChunkedData", func(ctx context.Context, h backend.Handler) {
		_ = h.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{}, nil)
	}},
	{"CallResource", func(ctx context.Context, h backend.Handler) {
		_ = h.CallResource(ctx, &backend.CallResourceRequest{}, backend.CallResourceResponseSenderFunc(func(*backend.CallResourceResponse) error { return nil }))
	}},
	{"CheckHealth", func(ctx context.Context, h backend.Handler) { _, _ = h.CheckHealth(ctx, &backend.CheckHealthRequest{}) }},
	{"SubscribeStream", func(ctx context.Context, h backend.Handler) {
		_, _ = h.SubscribeStream(ctx, &backend.SubscribeStreamRequest{})
	}},
	{"PublishStream", func(ctx context.Context, h backend.Handler) {
		_, _ = h.PublishStream(ctx, &backend.PublishStreamRequest{})
	}},
	{"RunStream", func(ctx context.Context, h backend.Handler) {
		_ = h.RunStream(ctx, &backend.RunStreamRequest{}, backend.NewStreamSender(nil))
	}},
	{"ValidateAdmission", func(ctx context.Context, h backend.Handler) {
		_, _ = h.ValidateAdmission(ctx, &backend.AdmissionRequest{})
	}},
	{"MutateAdmission", func(ctx context.Context, h backend.Handler) {
		_, _ = h.MutateAdmission(ctx, &backend.AdmissionRequest{})
	}},
	{"ConvertObjects", func(ctx context.Context, h backend.Handler) {
		_, _ = h.ConvertObjects(ctx, &backend.ConversionRequest{})
	}},
}

func TestLicenseFromIncomingGRPCContext(t *testing.T) {
	info := licenseInfo{LicenseToken: "token", AppURL: "https://grafana.example.com", ValidationKeys: "keys"}
	encoded, err := json.Marshal(info)
	require.NoError(t, err)

	tests := []struct {
		name  string
		ctx   context.Context
		want  licenseInfo
		found bool
	}{
		{name: "no metadata", ctx: context.Background()},
		{name: "nil metadata", ctx: metadata.NewIncomingContext(context.Background(), nil)},
		{name: "missing header", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("other", "value"))},
		{name: "empty header", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", ""))},
		{name: "malformed JSON", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", "{"))},
		{name: "wrong field type", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", `{"licenseToken":123}`))},
		{name: "valid metadata", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", string(encoded))), want: info, found: true},
		{name: "first header wins", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", string(encoded), "license", "{")), want: info, found: true},
		{name: "no fallback to later header", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", "", "license", string(encoded)))},
		{name: "outgoing metadata ignored", ctx: metadata.NewOutgoingContext(context.Background(), metadata.Pairs("license", string(encoded)))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := licenseFromIncomingGRPCContext(tt.ctx)
			require.Equal(t, tt.found, found)
			require.Equal(t, tt.want, got)
		})
	}
}

// parseArgs are the arguments a licenseParser was called with.
type parseArgs struct{ token, appURL, validationKeys, pluginID string }

// fakeParser stands in for licensing.LoadTokenFromValue: the token string selects the result.
type fakeParser struct {
	calls []parseArgs
}

func (f *fakeParser) parse(token, appURL, validationKeys, pluginID string) *licensing.LicenseToken {
	f.calls = append(f.calls, parseArgs{token, appURL, validationKeys, pluginID})
	switch token {
	case "valid":
		return &licensing.LicenseToken{Status: licensing.Valid}
	case "grace":
		return &licensing.LicenseToken{Status: licensing.ExpiredInGracePeriod, Error: errors.New("token expired"), LicenseExpires: time.Now().Add(-24 * time.Hour).Unix()}
	case "past-grace":
		return &licensing.LicenseToken{Status: licensing.Expired, Error: errors.New("token expired"), LicenseExpires: time.Now().Add(-30 * 24 * time.Hour).Unix()}
	}
	return &licensing.LicenseToken{Status: licensing.Invalid, Error: errors.New("bad license")}
}

// licenseCtx returns a context with incoming gRPC metadata carrying token, like Grafana sends it.
func licenseCtx(t *testing.T, token string) context.Context {
	t.Helper()
	info, err := json.Marshal(licenseInfo{LicenseToken: token, AppURL: "http://grafana.test", ValidationKeys: "keys"})
	require.NoError(t, err)
	return metadata.NewIncomingContext(t.Context(), metadata.Pairs("license", string(info)))
}

func middlewareHandler(t *testing.T, parser *fakeParser) (backend.Handler, *recordingHandler) {
	t.Helper()
	final := &recordingHandler{}
	h, err := backend.HandlerFromMiddlewares(final, newLicenseMiddleware("child-plugin", "parent-app", parser.parse))
	require.NoError(t, err)
	return h, final
}

func TestLicenseMiddlewareRefusesAndForwards(t *testing.T) {
	for _, tt := range []struct {
		token     string
		forwarded bool
	}{
		{"valid", true},
		{"grace", true},
		{"past-grace", false},
		{"invalid", false},
	} {
		for _, call := range licensedCalls {
			t.Run(tt.token+"/"+call.name, func(t *testing.T) {
				h, final := middlewareHandler(t, &fakeParser{})
				call.invoke(licenseCtx(t, tt.token), h)
				if tt.forwarded {
					require.Equal(t, []string{call.name}, final.calls)
				} else {
					require.Empty(t, final.calls)
				}
			})
		}
	}
}

func TestLicenseMiddlewareCollectMetricsIsNotChecked(t *testing.T) {
	parser := &fakeParser{}
	h, final := middlewareHandler(t, parser)
	_, err := h.CollectMetrics(t.Context(), &backend.CollectMetricsRequest{})
	require.NoError(t, err)
	require.Equal(t, []string{"CollectMetrics"}, final.calls)
	require.Empty(t, parser.calls)
}

func TestLicenseMiddlewareRevalidatesEveryRequest(t *testing.T) {
	parser := &fakeParser{}
	h, final := middlewareHandler(t, parser)

	for _, tt := range []struct {
		token       string
		wantRefused bool
	}{{"valid", false}, {"invalid", true}, {"valid", false}} {
		res, err := h.CheckHealth(licenseCtx(t, tt.token), &backend.CheckHealthRequest{})
		require.NoError(t, err)
		require.Equal(t, tt.wantRefused, res.Status == backend.HealthStatusError)
	}
	require.Len(t, parser.calls, 3)
	require.Len(t, final.calls, 2)
}

func TestLicenseMiddlewarePassesMetadataAndLicensePluginIDToParser(t *testing.T) {
	parser := &fakeParser{}
	h, _ := middlewareHandler(t, parser)

	// The request names the nested plugin; the license is for the parent.
	res, err := h.CheckHealth(licenseCtx(t, "invalid"), &backend.CheckHealthRequest{PluginContext: backend.PluginContext{PluginID: "child-plugin"}})
	require.NoError(t, err)
	require.Equal(t, []parseArgs{{"invalid", "http://grafana.test", "keys", "parent-app"}}, parser.calls)
	require.Contains(t, string(res.JSONDetails), "The Marketplace plugin parent-app is not available")
}

func TestLicenseMiddlewareFailsClosedWithoutUsableMetadata(t *testing.T) {
	// Environment variables and license files must not be consulted.
	licenseFile := filepath.Join(t.TempDir(), "license.jwt")
	require.NoError(t, os.WriteFile(licenseFile, []byte("valid"), 0o600))
	t.Setenv(marketplaceLicenseTextEnv, "valid")
	t.Setenv(marketplaceLicensePathEnv, licenseFile)
	t.Setenv(marketplaceAppURLEnv, "http://localhost:3000")
	t.Setenv(marketplaceLicenseValidationKeyEnv, "keys")

	for _, tt := range []struct {
		name string
		ctx  context.Context
	}{
		{"no metadata", t.Context()},
		{"no license key", metadata.NewIncomingContext(t.Context(), metadata.Pairs("other", "x"))},
		{"unparsable metadata", metadata.NewIncomingContext(t.Context(), metadata.Pairs("license", "{not json"))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parser := &fakeParser{}
			h, final := middlewareHandler(t, parser)
			res, err := h.CheckHealth(tt.ctx, &backend.CheckHealthRequest{})
			require.NoError(t, err)
			require.Equal(t, backend.HealthStatusError, res.Status)
			require.Contains(t, string(res.JSONDetails), errNoLicenseFound.Error())
			require.Empty(t, parser.calls)
			require.Empty(t, final.calls)
		})
	}
}

type passthroughMiddleware struct{}

func (passthroughMiddleware) CreateHandlerMiddleware(next backend.Handler) backend.Handler {
	return next
}

// countingConversion is a QueryConversionHandler that counts the calls that reach it.
type countingConversion struct{ calls int }

func (c *countingConversion) ConvertQueryDataRequest(context.Context, *backend.QueryDataRequest) (*backend.QueryConversionResponse, error) {
	c.calls++
	return &backend.QueryConversionResponse{}, nil
}

func TestApplyRequestTimeLicensePrependsMiddleware(t *testing.T) {
	other := passthroughMiddleware{}
	in := backend.ServeOpts{HandlerMiddlewares: []backend.HandlerMiddleware{other}}

	got, err := applyRequestTimeLicenseWithDeps("child-plugin", "parent-app", in, (&fakeParser{}).parse)
	require.NoError(t, err)
	require.Len(t, got.HandlerMiddlewares, 2)
	_, isFunc := got.HandlerMiddlewares[0].(backend.HandlerMiddlewareFunc)
	require.True(t, isFunc)
	require.Equal(t, other, got.HandlerMiddlewares[1])
}

func setQueryTimeLicenseValidation(t *testing.T, enabled bool) {
	t.Helper()
	old := *queryTimeLicenseValidation
	*queryTimeLicenseValidation = enabled
	t.Cleanup(func() { *queryTimeLicenseValidation = old })
}

// Tests using setQueryTimeLicenseValidation must not run in parallel.

func TestLicensingModeSelection(t *testing.T) {
	for _, tt := range []struct {
		name    string
		flag    bool
		env     string
		request bool
	}{
		{"neither", false, "", false},
		{"flag only", true, "", true},
		{"env only", false, "request", true},
		{"flag and env", true, "request", true},
		{"other env value", false, "startup", false},
		{"flag with other env", true, "startup", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// In startup mode this replaces every handler with invalidLicenseHandler.
			t.Setenv(marketplaceLicenseTextEnv, "")
			t.Setenv(marketplaceLicensePathEnv, filepath.Join(t.TempDir(), "missing.jwt"))
			t.Setenv(marketplaceLicenseValidationModeEnv, tt.env)
			setQueryTimeLicenseValidation(t, tt.flag)

			final := &recordingHandler{}
			in := backend.ServeOpts{CheckHealthHandler: final, QueryDataHandler: final}
			got, err := Licensing(Opts{ParentPluginID: "parent-app"})("example-app", in)
			require.NoError(t, err)

			_, extraErr := Licensing(Opts{})("example-app", backend.ServeOpts{ExtraPlugins: plugin.PluginSet{"extra": nil}})
			if !tt.request {
				require.NoError(t, extraErr)
				_, ok := got.CheckHealthHandler.(*invalidLicenseHandler)
				require.True(t, ok, "CheckHealthHandler is %T", got.CheckHealthHandler)
				require.Empty(t, got.HandlerMiddlewares)
				return
			}
			require.ErrorContains(t, extraErr, "ExtraPlugins")
			require.Same(t, final, got.CheckHealthHandler)
			require.Same(t, final, got.QueryDataHandler)
			require.Len(t, got.HandlerMiddlewares, 1)
		})
	}
}

// TestRequestTimeLicenseGRPCServeOpts checks licensing through the gRPC adapters, including
// query conversion, which bypasses the normal handler middleware chain. Missing or invalid
// licenses must deny admission and both conversion paths without reaching plugin handlers.
func TestRequestTimeLicenseGRPCServeOpts(t *testing.T) {
	const licenseMsg = "The Marketplace plugin parent-app is not available"
	queryObject := &pluginv2.RawObject{Raw: []byte(`{"queries":[{"refId":"A"}]}`), ContentType: "application/json"}
	genericObject := &pluginv2.RawObject{Raw: []byte("x"), ContentType: "text/plain"}

	for _, tt := range []struct {
		name      string
		ctx       func(t *testing.T) context.Context
		forwarded bool
	}{
		{"no metadata", func(t *testing.T) context.Context { t.Helper(); return t.Context() }, false},
		{"invalid token", func(t *testing.T) context.Context { t.Helper(); return licenseCtx(t, "invalid") }, false},
		{"valid token", func(t *testing.T) context.Context { t.Helper(); return licenseCtx(t, "valid") }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			final := &recordingHandler{}
			conv := &countingConversion{}
			opts, err := applyRequestTimeLicenseWithDeps("child-plugin", "parent-app", backend.ServeOpts{
				AdmissionHandler:       final,
				ConversionHandler:      final,
				QueryConversionHandler: conv,
			}, (&fakeParser{}).parse)
			require.NoError(t, err)
			grpcOpts, err := backend.GRPCServeOpts(opts)
			require.NoError(t, err)

			ctx := tt.ctx(t)
			pluginCtx := &pluginv2.PluginContext{PluginId: "child-plugin"}
			admissionReq := &pluginv2.AdmissionRequest{PluginContext: pluginCtx, Kind: &pluginv2.GroupVersionKind{}}
			convert := func(obj *pluginv2.RawObject) (*pluginv2.ConversionResponse, error) {
				t.Helper()
				return grpcOpts.ConversionServer.ConvertObjects(ctx, &pluginv2.ConversionRequest{
					PluginContext: pluginCtx, TargetVersion: &pluginv2.GroupVersion{}, Objects: []*pluginv2.RawObject{obj},
				})
			}

			validate, err := grpcOpts.AdmissionServer.ValidateAdmission(ctx, admissionReq)
			require.NoError(t, err)
			mutate, err := grpcOpts.AdmissionServer.MutateAdmission(ctx, admissionReq)
			require.NoError(t, err)
			_, queryErr := convert(queryObject)
			_, genericErr := convert(genericObject)

			if tt.forwarded {
				require.True(t, validate.Allowed)
				require.True(t, mutate.Allowed)
				require.NoError(t, queryErr)
				require.NoError(t, genericErr)
				require.Equal(t, 1, conv.calls)
				require.Equal(t, []string{"ValidateAdmission", "MutateAdmission", "ConvertObjects"}, final.calls)
				return
			}
			for _, res := range []*pluginv2.StatusResult{validate.Result, mutate.Result} {
				require.NotNil(t, res)
				require.Equal(t, "Failure", res.Status)
				require.EqualValues(t, http.StatusForbidden, res.Code)
				require.Equal(t, "Forbidden", res.Reason)
				require.Contains(t, res.Message, licenseMsg)
			}
			require.False(t, validate.Allowed)
			require.False(t, mutate.Allowed)
			require.ErrorContains(t, queryErr, licenseMsg)
			require.ErrorContains(t, genericErr, licenseMsg)
			require.Zero(t, conv.calls)
			require.Empty(t, final.calls)
		})
	}
}
