package maputil

import (
	"reflect"
	"testing"
)

func TestGetMap(t *testing.T) {
	tests := []struct {
		name           string
		obj            map[string]interface{}
		key            string
		expectedResult map[string]interface{}
		expectedError  string
	}{
		{
			name: "ExistingKey",
			obj: map[string]interface{}{
				"key1": map[string]interface{}{
					"innerKey": "value",
				},
			},
			key:            "key1",
			expectedResult: map[string]interface{}{"innerKey": "value"},
			expectedError:  "",
		},
		{
			name:           "NonExistingKey",
			obj:            map[string]interface{}{},
			key:            "key1",
			expectedResult: nil,
			expectedError:  "the field 'key1' should be set",
		},
		{
			name: "InvalidType",
			obj: map[string]interface{}{
				"key1": "not an object",
			},
			key:            "key1",
			expectedResult: nil,
			expectedError:  "the field 'key1' should be an object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetMap(tt.obj, tt.key)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error '%s' but got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error '%s' but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}

				if !reflect.DeepEqual(result, tt.expectedResult) {
					t.Errorf("expected %v but got %v", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestGetMapOptional(t *testing.T) {
	tests := []struct {
		name           string
		obj            map[string]interface{}
		key            string
		expectedResult map[string]interface{}
		expectedError  string
	}{
		{
			name: "ExistingKeyMap",
			obj: map[string]interface{}{
				"key1": map[string]interface{}{
					"innerKey": "value",
				},
			},
			key:            "key1",
			expectedResult: map[string]interface{}{"innerKey": "value"},
			expectedError:  "",
		},
		{
			name:           "ExistingKeyNonMap",
			obj:            map[string]interface{}{"key1": "not an object"},
			key:            "key1",
			expectedResult: nil,
			expectedError:  "the field 'key1' should be an object",
		},
		{
			name:           "NonExistingKey",
			obj:            map[string]interface{}{},
			key:            "key1",
			expectedResult: nil,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetMapOptional(tt.obj, tt.key)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error '%s' but got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error '%s' but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}

				if !reflect.DeepEqual(result, tt.expectedResult) {
					t.Errorf("expected %v but got %v", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name           string
		obj            map[string]interface{}
		key            string
		expectedResult bool
		expectedError  string
	}{
		{
			name:           "ExistingKeyTrue",
			obj:            map[string]interface{}{"key1": true},
			key:            "key1",
			expectedResult: true,
			expectedError:  "",
		},
		{
			name:           "ExistingKeyFalse",
			obj:            map[string]interface{}{"key1": false},
			key:            "key1",
			expectedResult: false,
			expectedError:  "",
		},
		{
			name:           "NonExistingKey",
			obj:            map[string]interface{}{},
			key:            "key1",
			expectedResult: false,
			expectedError:  "the field 'key1' should be set",
		},
		{
			name:           "InvalidType",
			obj:            map[string]interface{}{"key1": "not a bool"},
			key:            "key1",
			expectedResult: false,
			expectedError:  "the field 'key1' should be a bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetBool(tt.obj, tt.key)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error '%s' but got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error '%s' but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}

				if result != tt.expectedResult {
					t.Errorf("expected %v but got %v", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestGetBoolOptional(t *testing.T) {
	tests := []struct {
		name           string
		obj            map[string]interface{}
		key            string
		expectedResult bool
		expectedError  string
	}{
		{
			name:           "ExistingKeyTrue",
			obj:            map[string]interface{}{"key1": true},
			key:            "key1",
			expectedResult: true,
			expectedError:  "",
		},
		{
			name:           "ExistingKeyFalse",
			obj:            map[string]interface{}{"key1": false},
			key:            "key1",
			expectedResult: false,
			expectedError:  "",
		},
		{
			name:           "NonExistingKey",
			obj:            map[string]interface{}{},
			key:            "key1",
			expectedResult: false,
			expectedError:  "",
		},
		{
			name:           "InvalidType",
			obj:            map[string]interface{}{"key1": "not a bool"},
			key:            "key1",
			expectedResult: false,
			expectedError:  "the field 'key1' should be a bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetBoolOptional(tt.obj, tt.key)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error '%s' but got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error '%s' but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}

				if result != tt.expectedResult {
					t.Errorf("expected %v but got %v", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name           string
		obj            map[string]interface{}
		key            string
		expectedResult string
		expectedError  string
	}{
		{
			name:           "ExistingKey",
			obj:            map[string]interface{}{"key1": "value"},
			key:            "key1",
			expectedResult: "value",
			expectedError:  "",
		},
		{
			name:           "NonExistingKey",
			obj:            map[string]interface{}{},
			key:            "key1",
			expectedResult: "",
			expectedError:  "the field 'key1' should be set",
		},
		{
			name:           "InvalidType",
			obj:            map[string]interface{}{"key1": 123},
			key:            "key1",
			expectedResult: "",
			expectedError:  "the field 'key1' should be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetString(tt.obj, tt.key)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error '%s' but got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error '%s' but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}

				if result != tt.expectedResult {
					t.Errorf("expected %v but got %v", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestGetStringOptional(t *testing.T) {
	tests := []struct {
		name           string
		obj            map[string]interface{}
		key            string
		expectedResult string
		expectedError  string
	}{
		{
			name:           "ExistingKey",
			obj:            map[string]interface{}{"key1": "value"},
			key:            "key1",
			expectedResult: "value",
			expectedError:  "",
		},
		{
			name:           "NonExistingKey",
			obj:            map[string]interface{}{},
			key:            "key1",
			expectedResult: "",
			expectedError:  "",
		},
		{
			name:           "InvalidType",
			obj:            map[string]interface{}{"key1": 123},
			key:            "key1",
			expectedResult: "",
			expectedError:  "the field 'key1' should be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetStringOptional(tt.obj, tt.key)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error '%s' but got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("expected error '%s' but got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}

				if result != tt.expectedResult {
					t.Errorf("expected %v but got %v", tt.expectedResult, result)
				}
			}
		})
	}
}
