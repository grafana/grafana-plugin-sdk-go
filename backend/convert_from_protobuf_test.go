package backend

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/mitchellh/reflectwalk"
	"github.com/stretchr/testify/require"
)

type walker struct {
	FieldCount          int64
	ZeroValueFieldCount int64
}

func (w *walker) Struct(v reflect.Value) error {
	return nil
}

func (w *walker) StructField(f reflect.StructField, v reflect.Value) error {
	if strings.HasPrefix(f.Name, "XXX") {
		return nil
	}

	if f.PkgPath != "" {
		return nil
	}
	w.FieldCount++
	if v.IsZero() {
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

func TestConvertFromProtobufAppInstanceSettings(t *testing.T) {
	protoAIS := &pluginv2.AppInstanceSettings{
		JsonData:                []byte(`{ "foo": "gpp"`),
		DecryptedSecureJsonData: map[string]string{"secret": "quiet"},
		LastUpdatedMS:           86400 * 2 * 1000,
	}
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
	requireCounter.Equal(t, time.Unix(0, 86400*2*1000000000), sdkAIS.Updated)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount, "untested fields in conversion")

}

func TestConvertFromProtobufDataSourceInstanceSettings(t *testing.T) {
	protoDSIS := &pluginv2.DataSourceInstanceSettings{
		Id:                      2,
		Name:                    "bestData",
		Url:                     "http://grafana.com",
		User:                    "aUser",
		Database:                "grafana",
		BasicAuthEnabled:        true,
		BasicAuthUser:           "anotherUser",
		JsonData:                []byte(`{ "foo": "gpp"`),
		DecryptedSecureJsonData: map[string]string{"secret": "quiet"},
		LastUpdatedMS:           86400 * 2 * 1000,
	}
	protoWalker := &walker{}
	err := reflectwalk.Walk(protoDSIS, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "proto", "DataSourceInstanceSettings", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkDSIS := f.DataSourceInstanceSettings(protoDSIS)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkDSIS, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "DataSourceInstanceSettings", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, protoDSIS.Id, sdkDSIS.ID)
	requireCounter.Equal(t, protoDSIS.Name, sdkDSIS.Name)
	requireCounter.Equal(t, protoDSIS.Url, sdkDSIS.URL)
	requireCounter.Equal(t, protoDSIS.User, sdkDSIS.User)
	requireCounter.Equal(t, protoDSIS.Database, sdkDSIS.Database)
	requireCounter.Equal(t, protoDSIS.BasicAuthEnabled, sdkDSIS.BasicAuthEnabled)
	requireCounter.Equal(t, protoDSIS.BasicAuthUser, sdkDSIS.BasicAuthUser)
	requireCounter.Equal(t, json.RawMessage(protoDSIS.JsonData), sdkDSIS.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkDSIS.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1000000000), sdkDSIS.Updated)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount, "untested fields in conversion")

}

func TestConvertFromProtobufPluginContext(t *testing.T) {
	protoCtx := &pluginv2.PluginContext{
		OrgId:    3,
		PluginId: "the-best-plugin",
		User: &pluginv2.User{
			Login: "bestUser",
			Name:  "Best User",
			Email: "example@justAstring",
			Role:  "Lord",
		},
		AppInstanceSettings: &pluginv2.AppInstanceSettings{
			JsonData:                []byte(`{ "foo": "gpp"`),
			DecryptedSecureJsonData: map[string]string{"secret": "quiet"},
			LastUpdatedMS:           86400 * 2 * 1000,
		},
		DataSourceInstanceSettings: &pluginv2.DataSourceInstanceSettings{
			Id:                      2,
			Name:                    "bestData",
			Url:                     "http://grafana.com",
			User:                    "aUser",
			Database:                "grafana",
			BasicAuthEnabled:        true,
			BasicAuthUser:           "anotherUser",
			JsonData:                []byte(`{ "foo": "gpp"`),
			DecryptedSecureJsonData: map[string]string{"secret": "quiet"},
			LastUpdatedMS:           86400 * 2 * 1000,
		},
	}
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

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

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
	requireCounter.Equal(t, time.Unix(0, 86400*2*1000000000), sdkCtx.AppInstanceSettings.Updated)

	// Datasource Instance Settings
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Name, sdkCtx.DataSourceInstanceSettings.Name)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Id, sdkCtx.DataSourceInstanceSettings.ID)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Url, sdkCtx.DataSourceInstanceSettings.URL)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.User, sdkCtx.DataSourceInstanceSettings.User)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.Database, sdkCtx.DataSourceInstanceSettings.Database)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.BasicAuthEnabled, sdkCtx.DataSourceInstanceSettings.BasicAuthEnabled)
	requireCounter.Equal(t, protoCtx.DataSourceInstanceSettings.BasicAuthUser, sdkCtx.DataSourceInstanceSettings.BasicAuthUser)
	requireCounter.Equal(t, json.RawMessage(protoCtx.DataSourceInstanceSettings.JsonData), sdkCtx.DataSourceInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkCtx.DataSourceInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1000000000), sdkCtx.DataSourceInstanceSettings.Updated)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount-3, "untested fields in conversion") // -3 Struct Fields

}

func TestConvertFromProtobufTimeRange(t *testing.T) {
	protoTR := &pluginv2.TimeRange{
		FromEpochMS: 86400 * 2 * 1000,
		ToEpochMS:   (86400*2+3600)*1000 + 123,
	}

	protoWalker := &walker{}
	err := reflectwalk.Walk(protoTR, protoWalker)
	require.NoError(t, err)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "User", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkTR := f.TimeRange(protoTR)

	sdkWalker := &walker{}
	err = reflectwalk.Walk(sdkTR, sdkWalker)
	require.NoError(t, err)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "User", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, time.Unix(0, 86400*2*1000000000), sdkTR.From)
	requireCounter.Equal(t, time.Unix(0, (86400*2+3600)*1000000000+123000000), sdkTR.To)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount, "untested fields in conversion")

}
