package maputil

import "fmt"

func GetMap(obj map[string]any, key string) (map[string]any, error) {
	if untypedValue, ok := obj[key]; ok {
		if value, ok := untypedValue.(map[string]any); ok {
			return value, nil
		} else {
			err := fmt.Errorf("the field '%s' should be an object", key)
			return nil, err
		}
	} else {
		err := fmt.Errorf("the field '%s' should be set", key)
		return nil, err
	}
}

func GetBool(obj map[string]any, key string) (bool, error) {
	if untypedValue, ok := obj[key]; ok {
		if value, ok := untypedValue.(bool); ok {
			return value, nil
		} else {
			err := fmt.Errorf("the field '%s' should be a bool", key)
			return false, err
		}
	} else {
		err := fmt.Errorf("the field '%s' should be set", key)
		return false, err
	}
}

func GetBoolOptional(obj map[string]any, key string) (bool, error) {
	if untypedValue, ok := obj[key]; ok {
		if value, ok := untypedValue.(bool); ok {
			return value, nil
		} else {
			err := fmt.Errorf("the field '%s' should be a bool", key)
			return false, err
		}
	} else {
		// Value optional, not error
		return false, nil
	}
}

func GetString(obj map[string]any, key string) (string, error) {
	if untypedValue, ok := obj[key]; ok {
		if value, ok := untypedValue.(string); ok {
			return value, nil
		} else {
			err := fmt.Errorf("the field '%s' should be a string", key)
			return "", err
		}
	} else {
		err := fmt.Errorf("the field '%s' should be set", key)
		return "", err
	}
}

func GetStringOptional(obj map[string]any, key string) (string, error) {
	if untypedValue, ok := obj[key]; ok {
		if value, ok := untypedValue.(string); ok {
			return value, nil
		} else {
			err := fmt.Errorf("the field '%s' should be a string", key)
			return "", err
		}
	} else {
		// Value optional, not error
		return "", nil
	}
}
