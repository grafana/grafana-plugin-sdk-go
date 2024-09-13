package backend

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/stretchr/testify/require"
)

func TestConvertObjects(t *testing.T) {
	t.Run("converts a raw object", func(t *testing.T) {
		input := []byte(`{"foo":"bar"}`)
		expected := []byte(`{"baz":"qux"}`)

		a := newConversionSDKAdapter(ConvertObjectsFunc(func(_ context.Context, r *ConversionRequest) (*ConversionResponse, error) {
			require.Equal(t, input, r.Objects[0].Raw)
			return &ConversionResponse{
				Objects: []RawObject{{Raw: expected}},
			}, nil
		}), nil)
		res, err := a.ConvertObjects(context.Background(), &pluginv2.ConversionRequest{
			PluginContext: &pluginv2.PluginContext{},
			TargetVersion: &pluginv2.GroupVersion{},
			Objects:       []*pluginv2.RawObject{{Raw: input}},
		})
		require.NoError(t, err)
		require.Equal(t, []*pluginv2.RawObject{{Raw: expected}}, res.Objects)
	})

	t.Run("converts a query data request", func(t *testing.T) {
		input := []byte(`{"queries":[{"JSON":"foo"}]}`)

		a := newConversionSDKAdapter(nil, ConvertQueryFunc(func(_ context.Context, r *QueryDataRequest) (*QueryConversionResponse, error) {
			require.Equal(t, `"foo"`, string(r.Queries[0].JSON))
			return &QueryConversionResponse{
				Queries: []any{DataQuery{JSON: []byte(`"bar"`)}},
			}, nil
		}))
		res, err := a.ConvertObjects(context.Background(), &pluginv2.ConversionRequest{
			PluginContext: &pluginv2.PluginContext{},
			TargetVersion: &pluginv2.GroupVersion{},
			Objects:       []*pluginv2.RawObject{{Raw: input, ContentType: "application/json"}},
		})
		require.NoError(t, err)
		require.Contains(t, string(res.Objects[0].Raw), `"JSON":"bar"`)
	})
}
