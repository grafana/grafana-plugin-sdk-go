package backend

import (
	"encoding/json"
	"fmt"
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
		fmt.Println(f)
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
