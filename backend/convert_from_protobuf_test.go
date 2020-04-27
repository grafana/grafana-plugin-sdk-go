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
	reflectwalk.Walk(protoUser, protoWalker)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt,
			"proto", "User", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkUser := f.User(protoUser)

	sdkWalker := &walker{}
	reflectwalk.Walk(sdkUser, sdkWalker)

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
	protoAppInstanceSettings := &pluginv2.AppInstanceSettings{
		JsonData:                []byte(`{ "foo": "gpp"`),
		DecryptedSecureJsonData: map[string]string{"secret": "quiet"},
		LastUpdatedMS:           86400 * 2 * 1000,
	}
	protoWalker := &walker{}
	reflectwalk.Walk(protoAppInstanceSettings, protoWalker)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "proto", "AppInstanceSettings", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkAppInstanceSettings := f.AppInstanceSettings(protoAppInstanceSettings)

	sdkWalker := &walker{}
	reflectwalk.Walk(sdkAppInstanceSettings, sdkWalker)

	if sdkWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "sdk", "AppInstanceSettings", sdkWalker.ZeroValueFieldCount, sdkWalker.FieldCount)
	}

	require.Equal(t, protoWalker.FieldCount, sdkWalker.FieldCount)

	requireCounter := &requireCounter{}

	requireCounter.Equal(t, json.RawMessage(protoAppInstanceSettings.JsonData), sdkAppInstanceSettings.JSONData)
	requireCounter.Equal(t, map[string]string{"secret": "quiet"}, sdkAppInstanceSettings.DecryptedSecureJSONData)
	requireCounter.Equal(t, time.Unix(0, 86400*2*1000*1000000), sdkAppInstanceSettings.Updated)

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
	reflectwalk.Walk(protoDSIS, protoWalker)

	if protoWalker.HasZeroFields() {
		t.Fatalf(unsetErrFmt, "proto", "DataSourceInstanceSettings", protoWalker.ZeroValueFieldCount, protoWalker.FieldCount)
	}

	sdkDSIS := f.DataSourceInstanceSettings(protoDSIS)

	sdkWalker := &walker{}
	reflectwalk.Walk(sdkDSIS, sdkWalker)

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
	requireCounter.Equal(t, time.Unix(0, 86400*2*1000*1000000), sdkDSIS.Updated)

	require.Equal(t, requireCounter.Count, sdkWalker.FieldCount, "untested fields in conversion")

}
