package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mitchellh/reflectwalk"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type walker struct {
	FieldCount          int64
	ZeroValueFieldCount int64
}

func (w *walker) Struct(_ reflect.Value) error {
	return nil
}

func (w *walker) StructField(f reflect.StructField, v reflect.Value) error {
	if f.Anonymous {
		return nil
	}

	if strings.HasPrefix(f.Name, "XXX") {
		return nil
	}

	if f.PkgPath != "" {
		return nil
	}
	w.FieldCount++
	if v.IsZero() {
		fmt.Printf("Zero value field: %s\n", f.Name)
		w.ZeroValueFieldCount++
	}
	return nil
}

func (w *walker) HasZeroFields() bool {
	return w.ZeroValueFieldCount != 0
}

type requireCounter struct {
	Count int64
}

func (rec *requireCounter) Equal(t *testing.T, expected, actual interface{}, msgAngArgs ...interface{}) {
	t.Helper()
	require.Equal(t, expected, actual, msgAngArgs...)
	rec.Count++
}

var f ConvertFromProtobuf

const unsetErrFmt = "%v type for %v has unset fields, %v of %v unset, set all fields for the test"

func TestConvertFromProtobufUser(t *testing.T) {
	protoUser := &pluginv2.User{
		Login: "bestUser",
		Name:  "Best User",
		Email: "example@justAstring",
		Role:  "Lord",
	}

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoUser, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "User", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkUser := f.User(protoUser)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkUser, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "User", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, protoUser.Login, sdkUser.Login)
	requireCounter.Equal(t, protoUser.Name, sdkUser.Name)
	requireCounter.Equal(t, protoUser.Email, sdkUser.Email)
	requireCounter.Equal(t, protoUser.Role, sdkUser.Role)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount, "untested fields in conversion")
}

var lastUpdatedMS int64 = 86400 * 2 * 1000
var lastUpdatedTime = time.Unix(0, 86400*2*1e9)

var protoAppInstanceSettings = &pluginv2.AppInstanceSettings{
	JsonData:                []byte(`{ "foo": "gpp"`),
	DecryptedSecureJsonData: map[string]string{"secret": "quiet"},
	LastUpdatedMS:           lastUpdatedMS,
	ApiVersion:              "v1beta2",
}

func TestConvertFromProtobufAppInstanceSettings(t *testing.T) {
	protoAIS := protoAppInstanceSettings
	protoWalker := &walker{}
	err := reflectwalk.Walk(protoAIS, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "proto", "AppInstanceSettings", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkAIS := f.AppInstanceSettings(protoAIS)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkAIS, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "AppInstanceSettings", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, json.RawMessage(protoAIS.JsonData), sdkAIS.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkAIS.DecryptedSecureJSONData)
	requireCounter.Equal(t, lastUpdatedTime, sdkAIS.Updated)
	requireCounter.Equal(t, protoAIS.ApiVersion, sdkAIS.APIVersion)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount, "untested fields in conversion")
}

var protoDataSourceInstanceSettings = &pluginv2.DataSourceInstanceSettings{
	Id:                      2,
	Uid:                     "uid 2",
	Name:                    "bestData",
	Url:                     "http://grafana.com",
	User:                    "aUser",
	Database:                "grafana",
	BasicAuthEnabled:        true,
	BasicAuthUser:           "anotherUser",
	JsonData:                []byte(`{ "foo": "gpp"`),
	DecryptedSecureJsonData: map[string]string{"secret": "quiet"},
	LastUpdatedMS:           lastUpdatedMS,
	ApiVersion:              "v0alpha3",
}

func TestConvertFromProtobufDataSourceInstanceSettings(t *testing.T) {
	protoDSIS := protoDataSourceInstanceSettings
	protoWalker := &walker{}
	err := reflectwalk.Walk(protoDSIS, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "proto", "DataSourceInstanceSettings", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkDSIS := f.DataSourceInstanceSettings(protoDSIS, "example-datasource")

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkDSIS, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "DataSourceInstanceSettings", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount+datasourceInstanceProtoFieldCountDelta(), sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, protoDSIS.Id, sdkDSIS.ID)
	requireCounter.Equal(t, protoDSIS.Uid, sdkDSIS.UID)
	requireCounter.Equal(t, "example-datasource", sdkDSIS.Type)
	requireCounter.Equal(t, protoDSIS.Name, sdkDSIS.Name)
	requireCounter.Equal(t, protoDSIS.Url, sdkDSIS.URL)
	requireCounter.Equal(t, protoDSIS.User, sdkDSIS.User)
	requireCounter.Equal(t, protoDSIS.Database, sdkDSIS.Database)
	requireCounter.Equal(t, protoDSIS.BasicAuthEnabled, sdkDSIS.BasicAuthEnabled)
	requireCounter.Equal(t, protoDSIS.BasicAuthUser, sdkDSIS.BasicAuthUser)
	requireCounter.Equal(t, json.RawMessage(protoDSIS.JsonData), sdkDSIS.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkDSIS.DecryptedSecureJSONData)
	requireCounter.Equal(t, lastUpdatedTime, sdkDSIS.Updated)
	requireCounter.Equal(t, protoDSIS.ApiVersion, sdkDSIS.APIVersion)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount, "untested fields in conversion")
}

var protoPluginContext = &pluginv2.PluginContext{
	OrgId:         3,
	PluginId:      "the-best-plugin",
	PluginVersion: "1.0.0",
	User: &pluginv2.User{
		Login: "bestUser",
		Name:  "Best User",
		Email: "example@justAstring",
		Role:  "Lord",
	},
	AppInstanceSettings:        protoAppInstanceSettings,
	DataSourceInstanceSettings: protoDataSourceInstanceSettings,
	GrafanaConfig: map[string]string{
		"foo": "bar",
	},
	UserAgent:  "Grafana/10.0.0 (linux; amd64)",
	ApiVersion: "v0alpha1",
}

func TestConvertFromProtobufPluginContext(t *testing.T) {
	protoCtx := protoPluginContext
	protoWalker := &walker{}
	err := reflectwalk.Walk(protoCtx, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "proto", "PluginContext", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkCtx := f.PluginContext(protoCtx)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkCtx, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "DataSourceInstanceSettings", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount+datasourceInstanceProtoFieldCountDelta(), sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, protoCtx.OrgId, sdkCtx.OrgID)
	requireCounter.Equal(t, protoCtx.PluginId, sdkCtx.PluginID)

	// User
	requireCounter.Equal(t, protoCtx.User.Login, sdkCtx.User.Login)
	requireCounter.Equal(t, protoCtx.User.Name, sdkCtx.User.Name)
	requireCounter.Equal(t, protoCtx.User.Email, sdkCtx.User.Email)
	requireCounter.Equal(t, protoCtx.User.Role, sdkCtx.User.Role)

	// App Instance Settings
	requireCounter.Equal(t, json.RawMessage(protoCtx.AppInstanceSettings.JsonData), sdkCtx.AppInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkCtx.AppInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1e9), sdkCtx.AppInstanceSettings.Updated)
	requireCounter.Equal(t, protoCtx.AppInstanceSettings.ApiVersion, sdkCtx.AppInstanceSettings.APIVersion)

	// Datasource Instance Settings
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Name, sdkCtx.DataSourceInstanceSettings.Name)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Id, sdkCtx.DataSourceInstanceSettings.ID)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Uid, sdkCtx.DataSourceInstanceSettings.UID)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.ApiVersion, sdkCtx.DataSourceInstanceSettings.APIVersion)
	requireCounter.Equal(t, protoCtx.PluginId, sdkCtx.DataSourceInstanceSettings.Type)
	requireCounter.Equal(t, protoCtx.PluginVersion, sdkCtx.PluginVersion)
	requireCounter.Equal(t, protoCtx.ApiVersion, sdkCtx.APIVersion)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Url, sdkCtx.DataSourceInstanceSettings.URL)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.User, sdkCtx.DataSourceInstanceSettings.User)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Database, sdkCtx.DataSourceInstanceSettings.Database)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.BasicAuthEnabled, sdkCtx.DataSourceInstanceSettings.BasicAuthEnabled)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.BasicAuthUser, sdkCtx.DataSourceInstanceSettings.BasicAuthUser)
	requireCounter.Equal(t, json.RawMessage(protoCtx.DataSourceInstanceSettings.JsonData), sdkCtx.DataSourceInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkCtx.DataSourceInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1e9), sdkCtx.DataSourceInstanceSettings.Updated)
	requireCounter.Equal(t, protoCtx.UserAgent, sdkCtx.UserAgent.String())

	requireCounter.Equal(t, protoCtx.GrafanaConfig, sdkCtx.GrafanaConfig.config)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount-3, "untested fields in conversion") // -3 Struct Fields
}

var protoTimeRange = &pluginv2.TimeRange{
	FromEpochMS: 86400 * 2 * 1000,
	ToEpochMS:   (86400*2+3600)*1000 + 123,
}

var sdkTimeRange = TimeRange{
	From: time.Unix(0, 86400*2*1e9),
	To:   time.Unix(0, (86400*2+3600)*1e9+1.23*1e8),
}

func TestConvertFromProtobufTimeRange(t *testing.T) {
	protoTR := protoTimeRange

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoTR, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "TimeRange", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkTR := f.TimeRange(protoTR)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkTR, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "TimeRange", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, sdkTimeRange.From, sdkTR.From)
	requireCounter.Equal(t, sdkTimeRange.To, sdkTR.To)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount, "untested fields in conversion")
}

var protoDataQuery = &pluginv2.DataQuery{
	RefId:         "Z",
	MaxDataPoints: 1e6,
	TimeRange:     protoTimeRange,
	IntervalMS:    60 * 1000,
	Json:          []byte(`{ "query": "SELECT * from FUN"`),
	QueryType:     "qt",
}

func TestConvertFromProtobufDataQuery(t *testing.T) {
	protoDQ := protoDataQuery

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoDQ, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "DataQuery", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkDQ := f.DataQuery(protoDQ)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkDQ, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "DataQuery", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, protoDQ.RefId, sdkDQ.RefID)
	requireCounter.Equal(t, protoDQ.MaxDataPoints, sdkDQ.MaxDataPoints)
	requireCounter.Equal(t, protoDQ.QueryType, sdkDQ.QueryType)

	requireCounter.Equal(t, time.Minute, sdkDQ.Interval)
	requireCounter.Equal(t, sdkTimeRange.From, sdkDQ.TimeRange.From)
	requireCounter.Equal(t, sdkTimeRange.To, sdkDQ.TimeRange.To)
	requireCounter.Equal(t, json.RawMessage(protoDQ.Json), sdkDQ.JSON)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount-1, "untested fields in conversion") // -1 Struct Fields
}

func TestConvertFromProtobufQueryDataRequest(t *testing.T) {
	protoQDR := &pluginv2.QueryDataRequest{
		PluginContext: protoPluginContext,
		Headers:       map[string]string{"SET-WIN": "GOAL!"},
		Queries: []*pluginv2.DataQuery{
			protoDataQuery,
		},
	}

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoQDR, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "QueryDataRequest", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkQDR := f.QueryDataRequest(protoQDR)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkQDR, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "QueryDataRequest", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount+datasourceInstanceProtoFieldCountDelta(), sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, protoQDR.Headers, sdkQDR.Headers)

	// PluginContext
	requireCounter.Equal(t, protoQDR.PluginContext.OrgId, sdkQDR.PluginContext.OrgID)
	requireCounter.Equal(t, protoQDR.PluginContext.PluginId, sdkQDR.PluginContext.PluginID)
	requireCounter.Equal(t, protoQDR.PluginContext.ApiVersion, sdkQDR.PluginContext.APIVersion)
	// User
	requireCounter.Equal(t, protoQDR.PluginContext.User.Login, sdkQDR.PluginContext.User.Login)
	requireCounter.Equal(t, protoQDR.PluginContext.User.Name, sdkQDR.PluginContext.User.Name)
	requireCounter.Equal(t, protoQDR.PluginContext.User.Email, sdkQDR.PluginContext.User.Email)
	requireCounter.Equal(t, protoQDR.PluginContext.User.Role, sdkQDR.PluginContext.User.Role)

	// App Instance Settings
	requireCounter.Equal(t, json.RawMessage(protoQDR.PluginContext.AppInstanceSettings.JsonData), sdkQDR.PluginContext.AppInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkQDR.PluginContext.AppInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1e9), sdkQDR.PluginContext.AppInstanceSettings.Updated)
	requireCounter.Equal(t, protoQDR.PluginContext.AppInstanceSettings.ApiVersion, sdkQDR.PluginContext.AppInstanceSettings.APIVersion)

	// Datasource Instance Settings
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.Name, sdkQDR.PluginContext.DataSourceInstanceSettings.Name)
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.Id, sdkQDR.PluginContext.DataSourceInstanceSettings.ID)
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.Uid, sdkQDR.PluginContext.DataSourceInstanceSettings.UID)
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.ApiVersion, sdkQDR.PluginContext.DataSourceInstanceSettings.APIVersion)
	requireCounter.Equal(t, protoQDR.PluginContext.PluginId, sdkQDR.PluginContext.DataSourceInstanceSettings.Type)
	requireCounter.Equal(t, protoQDR.PluginContext.PluginVersion, sdkQDR.PluginContext.PluginVersion)
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.Url, sdkQDR.PluginContext.DataSourceInstanceSettings.URL)
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.User, sdkQDR.PluginContext.DataSourceInstanceSettings.User)
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.Database, sdkQDR.PluginContext.DataSourceInstanceSettings.Database)
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.BasicAuthEnabled, sdkQDR.PluginContext.DataSourceInstanceSettings.BasicAuthEnabled)
	requireCounter.Equal(t, protoQDR.PluginContext.DataSourceInstanceSettings.BasicAuthUser, sdkQDR.PluginContext.DataSourceInstanceSettings.BasicAuthUser)
	requireCounter.Equal(t, json.RawMessage(protoQDR.PluginContext.DataSourceInstanceSettings.JsonData), sdkQDR.PluginContext.DataSourceInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkQDR.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1e9), sdkQDR.PluginContext.DataSourceInstanceSettings.Updated)
	requireCounter.Equal(t, protoQDR.PluginContext.UserAgent, sdkQDR.PluginContext.UserAgent.String())

	// Queries
	requireCounter.Equal(t, protoQDR.Queries[0].RefId, sdkQDR.Queries[0].RefID)
	requireCounter.Equal(t, protoQDR.Queries[0].MaxDataPoints, sdkQDR.Queries[0].MaxDataPoints)
	requireCounter.Equal(t, protoQDR.Queries[0].QueryType, sdkQDR.Queries[0].QueryType)
	requireCounter.Equal(t, time.Minute, sdkQDR.Queries[0].Interval)
	requireCounter.Equal(t, sdkTimeRange.From, sdkQDR.Queries[0].TimeRange.From)
	requireCounter.Equal(t, sdkTimeRange.To, sdkQDR.Queries[0].TimeRange.To)
	requireCounter.Equal(t, json.RawMessage(protoQDR.Queries[0].Json), sdkQDR.Queries[0].JSON)

	// -7 is:
	// PluginContext, .User, .AppInstanceSettings, .DataSourceInstanceSettings
	// DataQuery, .TimeRange, .GrafanaConfig
	//
	//
	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount-7, "untested fields in conversion") // -6 Struct Fields
}

func TestConvertFromProtobufCheckHealthRequest(t *testing.T) {
	t.Run("Should convert provided headers", func(t *testing.T) {
		protoReq := &pluginv2.CheckHealthRequest{
			PluginContext: protoPluginContext,
			Headers: map[string]string{
				"foo": "fooVal",
				"bar": "barVal",
			},
		}

		req := FromProto().CheckHealthRequest(protoReq)
		require.NotNil(t, req)
		require.NotNil(t, req.PluginContext)
		require.Equal(t, protoPluginContext.OrgId, req.PluginContext.OrgID)
		require.Equal(t, protoReq.Headers, req.Headers)
	})

	t.Run("Should handle nil-provided headers", func(t *testing.T) {
		protoReq := &pluginv2.CheckHealthRequest{
			PluginContext: protoPluginContext,
		}

		req := FromProto().CheckHealthRequest(protoReq)
		require.NotNil(t, req)
		require.Equal(t, map[string]string{}, req.Headers)
	})
}

func TestConvertFromProtobufSubscribeStreamRequest(t *testing.T) {
	t.Run("Should convert provided headers", func(t *testing.T) {
		protoReq := &pluginv2.SubscribeStreamRequest{
			PluginContext: protoPluginContext,
			Headers: map[string]string{
				"foo": "fooVal",
				"bar": "barVal",
			},
		}

		req := FromProto().SubscribeStreamRequest(protoReq)
		require.NotNil(t, req)
		require.NotNil(t, req.PluginContext)
		require.Equal(t, protoPluginContext.OrgId, req.PluginContext.OrgID)
		require.Equal(t, protoReq.Headers, req.Headers)
	})

	t.Run("Should handle nil-provided headers", func(t *testing.T) {
		protoReq := &pluginv2.SubscribeStreamRequest{
			PluginContext: protoPluginContext,
		}

		req := FromProto().SubscribeStreamRequest(protoReq)
		require.NotNil(t, req)
		require.Equal(t, map[string]string{}, req.Headers)
	})
}

func TestConvertFromProtobufPublishStreamRequest(t *testing.T) {
	t.Run("Should convert provided headers", func(t *testing.T) {
		protoReq := &pluginv2.PublishStreamRequest{
			PluginContext: protoPluginContext,
			Headers: map[string]string{
				"foo": "fooVal",
				"bar": "barVal",
			},
		}

		req := FromProto().PublishStreamRequest(protoReq)
		require.NotNil(t, req)
		require.NotNil(t, req.PluginContext)
		require.Equal(t, protoPluginContext.OrgId, req.PluginContext.OrgID)
		require.Equal(t, protoReq.Headers, req.Headers)
	})

	t.Run("Should handle nil-provided headers", func(t *testing.T) {
		protoReq := &pluginv2.PublishStreamRequest{
			PluginContext: protoPluginContext,
		}

		req := FromProto().PublishStreamRequest(protoReq)
		require.NotNil(t, req)
		require.Equal(t, map[string]string{}, req.Headers)
	})
}

func TestConvertFromProtobufRunStreamRequest(t *testing.T) {
	t.Run("Should convert provided headers", func(t *testing.T) {
		protoReq := &pluginv2.RunStreamRequest{
			PluginContext: protoPluginContext,
			Headers: map[string]string{
				"foo": "fooVal",
				"bar": "barVal",
			},
		}

		req := FromProto().RunStreamRequest(protoReq)
		require.NotNil(t, req)
		require.NotNil(t, req.PluginContext)
		require.Equal(t, protoPluginContext.OrgId, req.PluginContext.OrgID)
		require.Equal(t, protoReq.Headers, req.Headers)
	})

	t.Run("Should handle nil-provided headers", func(t *testing.T) {
		protoReq := &pluginv2.RunStreamRequest{
			PluginContext: protoPluginContext,
		}

		req := FromProto().RunStreamRequest(protoReq)
		require.NotNil(t, req)
		require.Equal(t, map[string]string{}, req.Headers)
	})
}

func TestConvertFromProtobufDataResponse(t *testing.T) {
	t.Run("Should convert data query response", func(t *testing.T) {
		tcs := []struct {
			rsp                 *pluginv2.DataResponse
			expectedStatus      Status
			expectedErrorSource ErrorSource
		}{
			{
				rsp: &pluginv2.DataResponse{
					Status: http.StatusOK,
				},
				expectedStatus: StatusOK,
			}, {
				rsp: &pluginv2.DataResponse{
					Status: http.StatusFailedDependency,
				},
				expectedStatus: Status(424),
			}, {
				rsp: &pluginv2.DataResponse{
					Status: http.StatusInternalServerError,
					Error:  "foo",
				},
				expectedStatus: Status(500),
			},
			{
				rsp: &pluginv2.DataResponse{
					Status:      http.StatusInternalServerError,
					Error:       "foo",
					ErrorSource: string(ErrorSourceDownstream),
				},
				expectedStatus:      Status(500),
				expectedErrorSource: ErrorSourceDownstream,
			},
		}

		for _, tc := range tcs {
			rsp, err := FromProto().QueryDataResponse(&pluginv2.QueryDataResponse{
				Responses: map[string]*pluginv2.DataResponse{
					"A": tc.rsp,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, rsp)
			require.Equal(t, tc.expectedStatus, rsp.Responses["A"].Status)
			require.Equal(t, tc.expectedErrorSource, rsp.Responses["A"].ErrorSource)
		}
	})
}

// datasourceInstanceProtoFieldCountDelta returns the extra number of SDK fields that do not exist in the protobuf.
func datasourceInstanceProtoFieldCountDelta() int64 {
	// returning 1 to account for the Type field in the SDK that is not in the protobuf
	return int64(1)
}

func TestConvertFromProtobufAdmissionRequest(t *testing.T) {
	protoAR := &pluginv2.AdmissionRequest{
		PluginContext: protoPluginContext,
		Operation:     pluginv2.AdmissionRequest_UPDATE,
		Kind: &pluginv2.GroupVersionKind{
			Group:   "g",
			Version: "v",
			Kind:    "k",
		},
		ObjectBytes:    []byte(`{"hello": "world"}`),
		OldObjectBytes: []byte(`{"x": "y"}`),
	}

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoAR, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "AdmissionRequest", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkAR := f.AdmissionRequest(protoAR)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkAR, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "AdmissionRequest", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount+datasourceInstanceProtoFieldCountDelta(), sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	// PluginContext
	requireCounter.Equal(t, protoAR.PluginContext.OrgId, sdkAR.PluginContext.OrgID)
	requireCounter.Equal(t, protoAR.PluginContext.PluginId, sdkAR.PluginContext.PluginID)
	requireCounter.Equal(t, protoAR.PluginContext.ApiVersion, sdkAR.PluginContext.APIVersion)
	// User
	requireCounter.Equal(t, protoAR.PluginContext.User.Login, sdkAR.PluginContext.User.Login)
	requireCounter.Equal(t, protoAR.PluginContext.User.Name, sdkAR.PluginContext.User.Name)
	requireCounter.Equal(t, protoAR.PluginContext.User.Email, sdkAR.PluginContext.User.Email)
	requireCounter.Equal(t, protoAR.PluginContext.User.Role, sdkAR.PluginContext.User.Role)

	// App Instance Settings
	requireCounter.Equal(t, json.RawMessage(protoAR.PluginContext.AppInstanceSettings.JsonData), sdkAR.PluginContext.AppInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkAR.PluginContext.AppInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1e9), sdkAR.PluginContext.AppInstanceSettings.Updated)
	requireCounter.Equal(t, protoAR.PluginContext.AppInstanceSettings.ApiVersion, sdkAR.PluginContext.AppInstanceSettings.APIVersion)

	// Datasource Instance Settings
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.Name, sdkAR.PluginContext.DataSourceInstanceSettings.Name)
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.Id, sdkAR.PluginContext.DataSourceInstanceSettings.ID)
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.Uid, sdkAR.PluginContext.DataSourceInstanceSettings.UID)
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.ApiVersion, sdkAR.PluginContext.DataSourceInstanceSettings.APIVersion)
	requireCounter.Equal(t, protoAR.PluginContext.PluginId, sdkAR.PluginContext.DataSourceInstanceSettings.Type)
	requireCounter.Equal(t, protoAR.PluginContext.PluginVersion, sdkAR.PluginContext.PluginVersion)
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.Url, sdkAR.PluginContext.DataSourceInstanceSettings.URL)
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.User, sdkAR.PluginContext.DataSourceInstanceSettings.User)
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.Database, sdkAR.PluginContext.DataSourceInstanceSettings.Database)
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.BasicAuthEnabled, sdkAR.PluginContext.DataSourceInstanceSettings.BasicAuthEnabled)
	requireCounter.Equal(t, protoAR.PluginContext.DataSourceInstanceSettings.BasicAuthUser, sdkAR.PluginContext.DataSourceInstanceSettings.BasicAuthUser)
	requireCounter.Equal(t, json.RawMessage(protoAR.PluginContext.DataSourceInstanceSettings.JsonData), sdkAR.PluginContext.DataSourceInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkAR.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1e9), sdkAR.PluginContext.DataSourceInstanceSettings.Updated)
	requireCounter.Equal(t, protoAR.PluginContext.UserAgent, sdkAR.PluginContext.UserAgent.String())

	// The actual request values
	requireCounter.Equal(t, protoAR.Kind.Group, sdkAR.Kind.Group)
	requireCounter.Equal(t, protoAR.Kind.Version, sdkAR.Kind.Version)
	requireCounter.Equal(t, protoAR.Kind.Kind, sdkAR.Kind.Kind)
	requireCounter.Equal(t, int(protoAR.Operation), int(sdkAR.Operation))
	requireCounter.Equal(t, protoAR.ObjectBytes, sdkAR.ObjectBytes)
	requireCounter.Equal(t, protoAR.OldObjectBytes, sdkAR.OldObjectBytes)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount-6, "untested fields in conversion") // -6 Struct Fields
}

func TestConvertFromProtobufMutationResponse(t *testing.T) {
	protoRSP := &pluginv2.MutationResponse{
		Allowed: true,
		Result: &pluginv2.StatusResult{
			Status:  "A",
			Message: "M",
			Reason:  "bad",
			Code:    500,
		},
		Warnings:    []string{"hello"},
		ObjectBytes: []byte(`{"hello": "world"}`),
	}

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoRSP, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "MutationResponse", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkRSP := f.MutationResponse(protoRSP)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkRSP, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "MutationResponse", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	// PluginContext
	requireCounter.Equal(t, protoRSP.Allowed, sdkRSP.Allowed)
	requireCounter.Equal(t, protoRSP.ObjectBytes, sdkRSP.ObjectBytes)
	requireCounter.Equal(t, protoRSP.Warnings, sdkRSP.Warnings)
	requireCounter.Equal(t, protoRSP.Result.Code, sdkRSP.Result.Code)
	requireCounter.Equal(t, protoRSP.Result.Message, sdkRSP.Result.Message)
	requireCounter.Equal(t, protoRSP.Result.Reason, sdkRSP.Result.Reason)
	requireCounter.Equal(t, protoRSP.Result.Status, sdkRSP.Result.Status)

	require.Equal(t, sdkWalker.FieldCount-1, requireCounter.Count, "untested fields in conversion")
}

func TestConvertFromProtobufConversionRequest(t *testing.T) {
	protoCR := &pluginv2.ConversionRequest{
		PluginContext: protoPluginContext,
		Uid:           "uid",
		TargetVersion: &pluginv2.GroupVersion{
			Group:   "test.example.com",
			Version: "v1",
		},
		Objects: []*pluginv2.RawObject{{
			Raw:         []byte(`{"hello": "world"}`),
			ContentType: "application/json",
		}},
	}

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoCR, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "ConversionRequest", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkCR := f.ConversionRequest(protoCR)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkCR, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "ConversionRequest", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount+datasourceInstanceProtoFieldCountDelta(), sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	// PluginContext
	requireCounter.Equal(t, protoCR.PluginContext.OrgId, sdkCR.PluginContext.OrgID)
	requireCounter.Equal(t, protoCR.PluginContext.PluginId, sdkCR.PluginContext.PluginID)
	requireCounter.Equal(t, protoCR.PluginContext.ApiVersion, sdkCR.PluginContext.APIVersion)
	// User
	requireCounter.Equal(t, protoCR.PluginContext.User.Login, sdkCR.PluginContext.User.Login)
	requireCounter.Equal(t, protoCR.PluginContext.User.Name, sdkCR.PluginContext.User.Name)
	requireCounter.Equal(t, protoCR.PluginContext.User.Email, sdkCR.PluginContext.User.Email)
	requireCounter.Equal(t, protoCR.PluginContext.User.Role, sdkCR.PluginContext.User.Role)

	// App Instance Settings
	requireCounter.Equal(t, json.RawMessage(protoCR.PluginContext.AppInstanceSettings.JsonData), sdkCR.PluginContext.AppInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkCR.PluginContext.AppInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1e9), sdkCR.PluginContext.AppInstanceSettings.Updated)
	requireCounter.Equal(t, protoCR.PluginContext.AppInstanceSettings.ApiVersion, sdkCR.PluginContext.AppInstanceSettings.APIVersion)

	// Datasource Instance Settings
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.Name, sdkCR.PluginContext.DataSourceInstanceSettings.Name)
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.Id, sdkCR.PluginContext.DataSourceInstanceSettings.ID)
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.Uid, sdkCR.PluginContext.DataSourceInstanceSettings.UID)
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.ApiVersion, sdkCR.PluginContext.DataSourceInstanceSettings.APIVersion)
	requireCounter.Equal(t, protoCR.PluginContext.PluginId, sdkCR.PluginContext.DataSourceInstanceSettings.Type)
	requireCounter.Equal(t, protoCR.PluginContext.PluginVersion, sdkCR.PluginContext.PluginVersion)
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.Url, sdkCR.PluginContext.DataSourceInstanceSettings.URL)
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.User, sdkCR.PluginContext.DataSourceInstanceSettings.User)
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.Database, sdkCR.PluginContext.DataSourceInstanceSettings.Database)
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.BasicAuthEnabled, sdkCR.PluginContext.DataSourceInstanceSettings.BasicAuthEnabled)
	requireCounter.Equal(t, protoCR.PluginContext.DataSourceInstanceSettings.BasicAuthUser, sdkCR.PluginContext.DataSourceInstanceSettings.BasicAuthUser)
	requireCounter.Equal(t, json.RawMessage(protoCR.PluginContext.DataSourceInstanceSettings.JsonData), sdkCR.PluginContext.DataSourceInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkCR.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1e9), sdkCR.PluginContext.DataSourceInstanceSettings.Updated)
	requireCounter.Equal(t, protoCR.PluginContext.UserAgent, sdkCR.PluginContext.UserAgent.String())

	// The actual request values
	requireCounter.Equal(t, protoCR.TargetVersion.Group, sdkCR.TargetVersion.Group)
	requireCounter.Equal(t, protoCR.TargetVersion.Version, sdkCR.TargetVersion.Version)
	requireCounter.Equal(t, protoCR.Uid, sdkCR.UID)
	requireCounter.Equal(t, protoCR.Objects[0].Raw, sdkCR.Objects[0].Raw)
	requireCounter.Equal(t, protoCR.Objects[0].ContentType, sdkCR.Objects[0].ContentType)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount-7, "untested fields in conversion") // -7 Struct Fields
}

func TestConvertFromProtobufConversionResponse(t *testing.T) {
	protoCR := &pluginv2.ConversionResponse{
		Uid: "uid",
		Result: &pluginv2.StatusResult{
			Status:  "A",
			Message: "M",
			Reason:  "bad",
			Code:    500,
		},
		Objects: []*pluginv2.RawObject{{
			Raw:         []byte(`{"hello": "world"}`),
			ContentType: "application/json",
		}},
	}

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoCR, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "ConversionRequest", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkCR := f.ConversionResponse(protoCR)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkCR, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "ConversionResponse", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, protoCR.Uid, sdkCR.UID)
	requireCounter.Equal(t, protoCR.Objects[0].Raw, sdkCR.Objects[0].Raw)
	requireCounter.Equal(t, protoCR.Objects[0].ContentType, sdkCR.Objects[0].ContentType)
	requireCounter.Equal(t, protoCR.Result.Code, sdkCR.Result.Code)
	requireCounter.Equal(t, protoCR.Result.Message, sdkCR.Result.Message)
	requireCounter.Equal(t, protoCR.Result.Reason, sdkCR.Result.Reason)
	requireCounter.Equal(t, protoCR.Result.Status, sdkCR.Result.Status)
	requireCounter.Equal(t, len(protoCR.Objects), len(sdkCR.Objects))

	// -1 is for the Result pointer
	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount-1, "untested fields in conversion")
}
