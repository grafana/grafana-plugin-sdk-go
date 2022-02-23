package backend

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/stretchr/testify/require"
)

func TestConvertToProtobufQueryDataRespone(t *testing.T) {
	t.Run("No accept specified and no proxy initialized should return Arrow protobuf response", func(t *testing.T) {
		dr := testDataResponse(t)
		qdr := NewQueryDataResponse()
		qdr.Responses["A"] = dr
		require.Nil(t, qdr.ResponseProxy())

		protoRes, err := ToProto().QueryDataResponse(qdr, "")
		require.NoError(t, err)
		require.NotNil(t, protoRes)
		require.Equal(t, pluginv2.QueryDataResponse_ARROW, protoRes.DataType)
	})

	t.Run("Arrow accept specified and no proxy initialized should return Arrow protobuf response", func(t *testing.T) {
		dr := testDataResponse(t)
		qdr := NewQueryDataResponse()
		qdr.Responses["A"] = dr
		require.Nil(t, qdr.ResponseProxy())

		protoRes, err := ToProto().QueryDataResponse(qdr, string(DataResponseTypeArrow))
		require.NoError(t, err)
		require.NotNil(t, protoRes)
		require.Equal(t, pluginv2.QueryDataResponse_ARROW, protoRes.DataType)
	})

	t.Run("JSON accept specified and no proxy initialized should return JSON protobuf response", func(t *testing.T) {
		dr := testDataResponse(t)
		qdr := NewQueryDataResponse()
		qdr.Responses["A"] = dr
		require.Nil(t, qdr.ResponseProxy())

		protoRes, err := ToProto().QueryDataResponse(qdr, string(DataResponseTypeJSON))
		require.NoError(t, err)
		require.NotNil(t, protoRes)
		require.Equal(t, pluginv2.QueryDataResponse_JSON, protoRes.DataType)
		str := string(protoRes.Data)
		require.Equal(t, `{"results":{"A":{"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}}}`, str)
	})
}
