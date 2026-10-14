//go:build !marketplace_dev

package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
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

// licensedCall invokes one endpoint and reports whether the response is the license refusal.
type licensedCall struct {
	name   string
	invoke func(ctx context.Context, h backend.Handler) (refused bool, err error)
}

func refusedCalls() []licensedCall {
	isLicenseErr := func(err error) bool {
		return err != nil && strings.Contains(err.Error(), "is not available with your current subscription")
	}
	return []licensedCall{
		{"QueryData", func(ctx context.Context, h backend.Handler) (bool, error) {
			resp, err := h.QueryData(ctx, &backend.QueryDataRequest{Queries: []backend.DataQuery{{RefID: "A"}, {RefID: "B"}}})
			if err != nil || len(resp.Responses) != 2 {
				return false, err
			}
			return resp.Responses["A"].Status == backend.StatusUnauthorized && resp.Responses["B"].Status == backend.StatusUnauthorized, nil
		}},
		{"QueryChunkedData", func(ctx context.Context, h backend.Handler) (bool, error) {
			w := &recordingChunkedDataWriter{errors: map[string]error{}}
			err := h.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{Queries: []backend.DataQuery{{RefID: "A"}}}, w)
			return err == nil && w.errors["A"] != nil, err
		}},
		{"CallResource", func(ctx context.Context, h backend.Handler) (bool, error) {
			status := 0
			err := h.CallResource(ctx, &backend.CallResourceRequest{}, backend.CallResourceResponseSenderFunc(func(r *backend.CallResourceResponse) error {
				status = r.Status
				return nil
			}))
			return status == http.StatusForbidden, err
		}},
		{"CheckHealth", func(ctx context.Context, h backend.Handler) (bool, error) {
			res, err := h.CheckHealth(ctx, &backend.CheckHealthRequest{})
			return err == nil && res.Status == backend.HealthStatusError, err
		}},
		{"SubscribeStream", func(ctx context.Context, h backend.Handler) (bool, error) {
			res, err := h.SubscribeStream(ctx, &backend.SubscribeStreamRequest{})
			return err == nil && res.Status == backend.SubscribeStreamStatusPermissionDenied, err
		}},
		{"PublishStream", func(ctx context.Context, h backend.Handler) (bool, error) {
			res, err := h.PublishStream(ctx, &backend.PublishStreamRequest{})
			return err == nil && res.Status == backend.PublishStreamStatusPermissionDenied, err
		}},
		{"RunStream", func(ctx context.Context, h backend.Handler) (bool, error) {
			err := h.RunStream(ctx, &backend.RunStreamRequest{}, backend.NewStreamSender(nil))
			return isLicenseErr(err), nil
		}},
		{"ValidateAdmission", func(ctx context.Context, h backend.Handler) (bool, error) {
			_, err := h.ValidateAdmission(ctx, &backend.AdmissionRequest{})
			return isLicenseErr(err), nil
		}},
		{"MutateAdmission", func(ctx context.Context, h backend.Handler) (bool, error) {
			_, err := h.MutateAdmission(ctx, &backend.AdmissionRequest{})
			return isLicenseErr(err), nil
		}},
		{"ConvertObjects", func(ctx context.Context, h backend.Handler) (bool, error) {
			_, err := h.ConvertObjects(ctx, &backend.ConversionRequest{})
			return isLicenseErr(err), nil
		}},
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
		return &licensing.LicenseToken{Status: licensing.Expired, Error: errors.New("token expired"), LicenseExpires: time.Now().Add(-24 * time.Hour).Unix()}
	case "past-grace":
		return &licensing.LicenseToken{Status: licensing.Expired, Error: errors.New("token expired"), LicenseExpires: time.Now().Add(-30 * 24 * time.Hour).Unix()}
	}
	return &licensing.LicenseToken{Status: licensing.Invalid, Error: errors.New("bad license")}
}

// licenseCtx returns a context with incoming gRPC metadata carrying token, like Grafana sends it.
func licenseCtx(t *testing.T, token string) context.Context {
	t.Helper()
	info, err := json.Marshal(licensing.LicenseInfo{LicenseToken: token, AppURL: "http://grafana.test", ValidationKeys: "keys"})
	require.NoError(t, err)
	return metadata.NewIncomingContext(t.Context(), metadata.Pairs("license", string(info)))
}

func middlewareHandler(t *testing.T, parser *fakeParser) (backend.Handler, *recordingHandler) {
	t.Helper()
	final := &recordingHandler{}
	h, err := backend.HandlerFromMiddlewares(final, newLicenseMiddleware("child-plugin", "parent-app", requestLicenseLoader(parser.parse)))
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
		for _, call := range refusedCalls() {
			t.Run(tt.token+"/"+call.name, func(t *testing.T) {
				h, final := middlewareHandler(t, &fakeParser{})
				refused, err := call.invoke(licenseCtx(t, tt.token), h)
				require.NoError(t, err)
				require.Equal(t, !tt.forwarded, refused)
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

	t.Run("production parser rejects an unparsable token", func(t *testing.T) {
		token := requestLicenseLoader(licensing.LoadTokenFromValue)(licenseCtx(t, "nope"), "parent-app")
		require.Error(t, token.Error)
		require.NotEqual(t, licensing.Valid, token.Status)
	})
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

func TestApplyRequestTimeLicense(t *testing.T) {
	t.Run("middleware is first and caller slice is untouched", func(t *testing.T) {
		other := passthroughMiddleware{}
		in := backend.ServeOpts{HandlerMiddlewares: make([]backend.HandlerMiddleware, 1, 4)}
		in.HandlerMiddlewares[0] = other

		got, err := applyRequestTimeLicenseWithDeps("child-plugin", "parent-app", in, requestLicenseLoader((&fakeParser{}).parse))
		require.NoError(t, err)
		require.Len(t, got.HandlerMiddlewares, 2)
		require.Len(t, in.HandlerMiddlewares, 1)
		require.Equal(t, other, in.HandlerMiddlewares[0])
		require.Equal(t, other, got.HandlerMiddlewares[1])
		_, isFunc := got.HandlerMiddlewares[0].(backend.HandlerMiddlewareFunc)
		require.True(t, isFunc)
		require.Nil(t, got.QueryConversionHandler)
	})

	t.Run("query conversion is wrapped", func(t *testing.T) {
		parser := &fakeParser{}
		next := &countingConversion{}
		got, err := applyRequestTimeLicenseWithDeps("child-plugin", "parent-app",
			backend.ServeOpts{QueryConversionHandler: next}, requestLicenseLoader(parser.parse))
		require.NoError(t, err)

		_, err = got.QueryConversionHandler.ConvertQueryDataRequest(licenseCtx(t, "invalid"), &backend.QueryDataRequest{})
		require.ErrorContains(t, err, "The Marketplace plugin parent-app is not available")
		require.Zero(t, next.calls)

		_, err = got.QueryConversionHandler.ConvertQueryDataRequest(licenseCtx(t, "valid"), &backend.QueryDataRequest{})
		require.NoError(t, err)
		require.Equal(t, 1, next.calls)

		_, err = got.QueryConversionHandler.ConvertQueryDataRequest(t.Context(), &backend.QueryDataRequest{})
		require.ErrorContains(t, err, "is not available")
		require.Equal(t, 1, next.calls)

		require.Equal(t, []parseArgs{
			{"invalid", "http://grafana.test", "keys", "parent-app"},
			{"valid", "http://grafana.test", "keys", "parent-app"},
		}, parser.calls)
	})

	t.Run("extra plugins are rejected", func(t *testing.T) {
		_, err := applyRequestTimeLicenseWithDeps("p", "p", backend.ServeOpts{ExtraPlugins: plugin.PluginSet{"extra": nil}}, requestLicenseLoader((&fakeParser{}).parse))
		require.ErrorContains(t, err, "ExtraPlugins")
	})
}

func setQueryTimeLicenseValidation(t *testing.T, enabled bool) {
	t.Helper()
	old := *queryTimeLicenseValidation
	*queryTimeLicenseValidation = enabled
	t.Cleanup(func() { *queryTimeLicenseValidation = old })
}

// Tests using setQueryTimeLicenseValidation must not run in parallel.

func TestLicensingQueryTimeModeSkipsStartupCheck(t *testing.T) {
	// In startup mode this would replace every handler with invalidLicenseHandler.
	t.Setenv(marketplaceLicenseTextEnv, "")
	t.Setenv(marketplaceLicensePathEnv, filepath.Join(t.TempDir(), "missing.jwt"))
	setQueryTimeLicenseValidation(t, true)

	in := licensedServeOpts()
	in.ExtraPlugins = nil
	got, err := Licensing(Opts{ParentPluginID: "parent-app"})("example-app", in)
	require.NoError(t, err)
	require.Same(t, in.CheckHealthHandler, got.CheckHealthHandler)
	require.Len(t, got.HandlerMiddlewares, 1)

	_, err = Licensing(Opts{})("example-app", licensedServeOpts())
	require.ErrorContains(t, err, "ExtraPlugins")
}

func TestRequestTimeMiddlewareChain(t *testing.T) {
	datasourceOpts := func(h backend.Handler) backend.ServeOpts {
		return backend.ServeOpts{QueryDataHandler: h, CheckHealthHandler: h, StreamHandler: h}
	}
	appOpts := func(h backend.Handler) backend.ServeOpts {
		return backend.ServeOpts{CallResourceHandler: h, CheckHealthHandler: h, AdmissionHandler: h, ConversionHandler: h}
	}
	shapes := []struct {
		name  string
		opts  func(backend.Handler) backend.ServeOpts
		calls []string
	}{
		{"datasource", datasourceOpts, []string{"QueryData", "CheckHealth", "SubscribeStream"}},
		{"app", appOpts, []string{"CallResource", "CheckHealth", "ValidateAdmission", "ConvertObjects"}},
	}
	byName := map[string]licensedCall{}
	for _, c := range refusedCalls() {
		byName[c.name] = c
	}

	for _, shape := range shapes {
		for _, tt := range []struct {
			name      string
			token     string // empty: no metadata
			forwarded bool
			parsed    bool
		}{
			{"valid license", "valid", true, true},
			{"invalid license", "invalid", false, true},
			{"no metadata", "", false, false},
		} {
			t.Run(shape.name+"/"+tt.name, func(t *testing.T) {
				parser := &fakeParser{}
				final := &recordingHandler{}
				serveOpts, err := applyRequestTimeLicenseWithDeps("child-plugin", "parent-app", shape.opts(final), requestLicenseLoader(parser.parse))
				require.NoError(t, err)
				h, err := serveOpts.HandlerWithMiddlewares()
				require.NoError(t, err)

				ctx := t.Context()
				if tt.token != "" {
					ctx = licenseCtx(t, tt.token)
				}
				for _, name := range shape.calls {
					refused, err := byName[name].invoke(ctx, h)
					require.NoError(t, err, name)
					require.Equal(t, !tt.forwarded, refused, name)
				}
				if tt.parsed {
					require.Len(t, parser.calls, len(shape.calls))
				} else {
					require.Empty(t, parser.calls)
				}
				if tt.forwarded {
					require.Equal(t, shape.calls, final.calls)
				} else {
					require.Empty(t, final.calls)
				}
			})
		}
	}
}
