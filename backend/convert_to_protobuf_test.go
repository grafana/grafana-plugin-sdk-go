package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/mitchellh/reflectwalk"
	"github.com/stretchr/testify/require"
)

func TestConvertToProtobufQueryDataResponse(t *testing.T) {
	frames := data.Frames{data.NewFrame("test", data.NewField("test", nil, []int64{1}))}
	tcs := []struct {
		name        string
		err         error
		status      Status
		errorSource ErrorSource

		expectedStatus      int32
		expectedErrorSource string
	}{
		{
			name:           "If a HTTP Status code is used, use backend.Status equivalent status code",
			status:         http.StatusOK,
			expectedStatus: int32(StatusOK),
		},
		{
			name:           "If a backend.Status is used, use backend.Status int code",
			status:         StatusTooManyRequests,
			expectedStatus: int32(StatusTooManyRequests),
		},
		{
			name:           "syscall.ECONNREFUSED is inferred as a Status Bad Gateway",
			err:            syscall.ECONNREFUSED,
			expectedStatus: int32(StatusBadGateway),
		},
		{
			name:           "os.ErrDeadlineExceeded is inferred as a Status Timeout",
			err:            os.ErrDeadlineExceeded,
			expectedStatus: int32(StatusTimeout),
		},
		{
			name:           "fs.ErrPermission is inferred as a Status Unauthorized",
			err:            fs.ErrPermission,
			expectedStatus: int32(StatusUnauthorized),
		},
		{
			name:           "Custom error is inferred as a Status Unknown",
			err:            fmt.Errorf("some custom error"),
			expectedStatus: int32(StatusUnknown),
		},
		{
			name:           "A wrapped error is appropriately inferred",
			err:            fmt.Errorf("wrap 2: %w", fmt.Errorf("wrap 1: %w", os.ErrDeadlineExceeded)),
			expectedStatus: int32(StatusTimeout),
		},
		{
			name:                "ErrorSource is marshalled",
			err:                 errors.New("oh no"),
			status:              StatusBadGateway,
			errorSource:         ErrorSourceDownstream,
			expectedStatus:      int32(StatusBadGateway),
			expectedErrorSource: "downstream",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			protoRes := &QueryDataResponse{
				Responses: map[string]DataResponse{
					"A": {
						Frames:      frames,
						Error:       tc.err,
						Status:      tc.status,
						ErrorSource: tc.errorSource,
					},
				},
			}
			qdr, err := ToProto().QueryDataResponse(protoRes)
			require.NoError(t, err)
			require.NotNil(t, qdr)
			require.NotNil(t, qdr.Responses)
			resp := qdr.Responses["A"]
			require.Equal(t, tc.expectedStatus, resp.Status)
			require.Equal(t, tc.expectedErrorSource, resp.ErrorSource)
		})
	}
}

func TestConvertToProtobufStatus(t *testing.T) {
	ar := ToProto().StatusResult(&StatusResult{
		Status:  "a",
		Message: "b",
		Reason:  "c",
		Code:    234,
	})
	require.NotNil(t, ar)
	require.Equal(t, "a", ar.Status)
	require.Equal(t, "b", ar.Message)
	require.Equal(t, "c", ar.Reason)
	require.Equal(t, int32(234), ar.Code)
}

func TestInstanceSettingsConversion(t *testing.T) {
	t.Run("DataSource", func(t *testing.T) {
		before := &DataSourceInstanceSettings{
			URL:      "http://something",
			Updated:  time.Now(),
			User:     "u",
			JSONData: []byte(`{"hello": "world"}`),
			DecryptedSecureJSONData: map[string]string{
				"A": "B",
			},
		}
		wire, err := DataSourceInstanceSettingsToProtoBytes(before)
		require.NoError(t, err)
		after, err := DataSourceInstanceSettingsFromProto(wire, "")
		require.NoError(t, err)
		require.Equal(t, before.URL, after.URL)
		require.Equal(t, before.User, after.User)
		require.Equal(t, before.JSONData, after.JSONData)
		require.Equal(t, before.DecryptedSecureJSONData, after.DecryptedSecureJSONData)
	})

	t.Run("App", func(t *testing.T) {
		before := &AppInstanceSettings{
			Updated:  time.Now(),
			JSONData: []byte(`{"hello": "world"}`),
			DecryptedSecureJSONData: map[string]string{
				"A": "B",
			},
		}
		wire, err := AppInstanceSettingsToProtoBytes(before)
		require.NoError(t, err)
		after, err := AppInstanceSettingsFromProto(wire)
		require.NoError(t, err)
		require.Equal(t, before.JSONData, after.JSONData)
		require.Equal(t, before.DecryptedSecureJSONData, after.DecryptedSecureJSONData)
	})
}

var testUserAgent, _ = useragent.New("7.0.0", "darwin", "amd64")

var testPluginContext = PluginContext{
	OrgID:         3,
	PluginID:      "pluginID",
	PluginVersion: "1.0.0",
	User: &User{
		Login: "login",
		Name:  "name",
		Email: "email",
		Role:  "role",
	},
	AppInstanceSettings: &AppInstanceSettings{
		Updated:                 time.Unix(1, 0),
		JSONData:                []byte(`{"hello": "world"}`),
		DecryptedSecureJSONData: map[string]string{"secret": "quiet"},
		APIVersion:              "v1",
	},
	DataSourceInstanceSettings: &DataSourceInstanceSettings{
		ID:                      1,
		UID:                     "uid",
		Type:                    "pluginID",
		Name:                    "name",
		URL:                     "http://example.com",
		User:                    "user",
		Database:                "database",
		BasicAuthEnabled:        true,
		BasicAuthUser:           "user",
		JSONData:                json.RawMessage(`{"hello": "world"}`),
		DecryptedSecureJSONData: map[string]string{"secret": "quiet"},
		Updated:                 time.Unix(2, 0),
		APIVersion:              "v1",
	},
	GrafanaConfig: &GrafanaCfg{config: map[string]string{"key": "value"}},
	UserAgent:     testUserAgent,
	APIVersion:    "v1",
}

func TestConvertToProtobufConversionRequest(t *testing.T) {
	sdkCR := &ConversionRequest{
		PluginContext: testPluginContext,
		UID:           "uid",
		TargetVersion: GroupVersion{
			Group:   "group",
			Version: "version",
		},
		Objects: []RawObject{
			{
				Raw:         []byte("raw"),
				ContentType: "content-type",
			},
		},
	}
	sdkWalker := &walker{}
	err := reflectwalk.Walk(sdkCR, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "ConversionRequest", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	protoCR := ToProto().ConversionRequest(sdkCR)

	protoWalker := &walker{}
	err = reflectwalk.Walk(protoCR, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "ConversionRequest", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	require.Equal(t, sdkWalker.FieldCount, protoWalker.FieldCount+datasourceInstanceProtoFieldCountDelta())

	requireCounter := &requireCounter{}

	// PluginContext
	requireCounter.Equal(t, sdkCR.PluginContext.OrgID, protoCR.PluginContext.OrgId)
	requireCounter.Equal(t, sdkCR.PluginContext.PluginID, protoCR.PluginContext.PluginId)
	requireCounter.Equal(t, sdkCR.PluginContext.APIVersion, protoCR.PluginContext.ApiVersion)
	// User
	requireCounter.Equal(t, sdkCR.PluginContext.User.Login, protoCR.PluginContext.User.Login)
	requireCounter.Equal(t, sdkCR.PluginContext.User.Name, protoCR.PluginContext.User.Name)
	requireCounter.Equal(t, sdkCR.PluginContext.User.Email, protoCR.PluginContext.User.Email)
	requireCounter.Equal(t, sdkCR.PluginContext.User.Role, protoCR.PluginContext.User.Role)

	// App Instance Settings
	requireCounter.Equal(t, sdkCR.PluginContext.AppInstanceSettings.JSONData, json.RawMessage(protoCR.PluginContext.AppInstanceSettings.JsonData))
	requireCounter.Equal(t, sdkCR.PluginContext.AppInstanceSettings.DecryptedSecureJSONData["secret"], protoCR.PluginContext.AppInstanceSettings.DecryptedSecureJsonData["secret"])
	requireCounter.Equal(t, sdkCR.PluginContext.AppInstanceSettings.Updated.UnixMilli(), protoCR.PluginContext.AppInstanceSettings.LastUpdatedMS)
	requireCounter.Equal(t, sdkCR.PluginContext.AppInstanceSettings.APIVersion, protoCR.PluginContext.AppInstanceSettings.ApiVersion)

	// Datasource Instance Settings
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.Name, protoCR.PluginContext.DataSourceInstanceSettings.Name)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.ID, protoCR.PluginContext.DataSourceInstanceSettings.Id)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.UID, protoCR.PluginContext.DataSourceInstanceSettings.Uid)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.APIVersion, protoCR.PluginContext.DataSourceInstanceSettings.ApiVersion)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.Type, protoCR.PluginContext.PluginId)
	requireCounter.Equal(t, sdkCR.PluginContext.PluginVersion, protoCR.PluginContext.PluginVersion)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.URL, protoCR.PluginContext.DataSourceInstanceSettings.Url)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.User, protoCR.PluginContext.DataSourceInstanceSettings.User)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.Database, protoCR.PluginContext.DataSourceInstanceSettings.Database)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.BasicAuthEnabled, protoCR.PluginContext.DataSourceInstanceSettings.BasicAuthEnabled)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.BasicAuthUser, protoCR.PluginContext.DataSourceInstanceSettings.BasicAuthUser)
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.JSONData, json.RawMessage(protoCR.PluginContext.DataSourceInstanceSettings.JsonData))
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["secret"], protoCR.PluginContext.DataSourceInstanceSettings.DecryptedSecureJsonData["secret"])
	requireCounter.Equal(t, sdkCR.PluginContext.DataSourceInstanceSettings.Updated.UnixMilli(), protoCR.PluginContext.DataSourceInstanceSettings.LastUpdatedMS)
	requireCounter.Equal(t, sdkCR.PluginContext.UserAgent.String(), protoCR.PluginContext.UserAgent)

	// The actual request values
	requireCounter.Equal(t, sdkCR.TargetVersion.Group, protoCR.TargetVersion.Group)
	requireCounter.Equal(t, sdkCR.TargetVersion.Version, protoCR.TargetVersion.Version)
	requireCounter.Equal(t, sdkCR.UID, protoCR.Uid)
	requireCounter.Equal(t, sdkCR.Objects[0].Raw, protoCR.Objects[0].Raw)
	requireCounter.Equal(t, sdkCR.Objects[0].ContentType, protoCR.Objects[0].ContentType)
}

func TestConvertToProtobufConversionResponse(t *testing.T) {
	sdkCR := &ConversionResponse{
		UID: "uid",
		Result: &StatusResult{
			Status:  "status",
			Message: "message",
			Reason:  "reason",
			Code:    1,
		},
		Objects: []RawObject{
			{
				Raw:         []byte("raw"),
				ContentType: "content-type",
			},
		},
	}
	sdkWalker := &walker{}
	err := reflectwalk.Walk(sdkCR, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "ConversionResponse", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	protoCR := ToProto().ConversionResponse(sdkCR)

	protoWalker := &walker{}
	err = reflectwalk.Walk(protoCR, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "ConversionResponse", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	require.Equal(t, sdkWalker.FieldCount, protoWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, sdkCR.UID, protoCR.Uid)
	requireCounter.Equal(t, sdkCR.Result.Status, protoCR.Result.Status)
	requireCounter.Equal(t, sdkCR.Result.Message, protoCR.Result.Message)
	requireCounter.Equal(t, sdkCR.Result.Reason, protoCR.Result.Reason)
	requireCounter.Equal(t, sdkCR.Result.Code, protoCR.Result.Code)
	requireCounter.Equal(t, sdkCR.Objects[0].Raw, protoCR.Objects[0].Raw)
	requireCounter.Equal(t, sdkCR.Objects[0].ContentType, protoCR.Objects[0].ContentType)
}
