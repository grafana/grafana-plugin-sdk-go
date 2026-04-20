package pluginschema

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"

	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/yaml"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
)

func TestPluginSchema_IsZero(t *testing.T) {
	tests := []struct {
		name string
		s    *PluginSchema
		want bool
	}{
		{"nil", nil, true},
		{"empty", &PluginSchema{}, true},
		{"with SettingsSchema", &PluginSchema{SettingsSchema: &Settings{Spec: &spec.Schema{}}}, false},
		{"with SettingsExamples", &PluginSchema{SettingsExamples: &SettingsExamples{Examples: map[string]*spec3.Example{
			"x": {},
		}}}, false},
		{"with Routes", &PluginSchema{Routes: &Routes{Paths: map[string]*spec3.Path{
			"/proxy": {},
		}}}, false},
		{"with QueryTypes", &PluginSchema{QueryTypes: &sdkapi.QueryTypeDefinitionList{Items: []sdkapi.QueryTypeDefinition{{}}}}, false},
		{"with QueryExamples", &PluginSchema{QueryExamples: &sdkapi.QueryExamples{Examples: []sdkapi.QueryExample{{}}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.IsZero(); got != tt.want {
				t.Errorf("PluginSchema.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotExists(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"ErrNotExist", fs.ErrNotExist, true},
		{"string not exist", errors.New("file does not exist"), true},
		{"other error", errors.New("other"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotExists(tt.err); got != tt.want {
				t.Errorf("isNotExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		jsonOrYaml []byte
		obj        any
		wantErr    bool
	}{
		{
			name:       "valid yaml",
			jsonOrYaml: []byte(`key: value`),
			obj:        &map[string]string{},
			wantErr:    false,
		},
		{
			name:       "valid json",
			jsonOrYaml: []byte(`{"key": "value"}`),
			obj:        &map[string]string{},
			wantErr:    false,
		},
		{
			name:       "invalid",
			jsonOrYaml: []byte(`invalid: :`),
			obj:        &map[string]string{},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Load(tt.jsonOrYaml, tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToYAML(t *testing.T) {
	obj := map[string]string{"key": "value"}
	data, err := ToYAML(obj)
	if err != nil {
		t.Errorf("ToYAML() error = %v", err)
	}
	var result map[string]string
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Errorf("Unmarshal failed: %v", err)
	}
	if !reflect.DeepEqual(result, obj) {
		t.Errorf("ToYAML() round trip failed")
	}
}
