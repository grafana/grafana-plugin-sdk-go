package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

func GetStringValueFromJSON(fpath string, key string) (string, error) {
	byteValue, err := os.ReadFile(fpath)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		return "", err
	}
	executable := result[key]
	name, ok := executable.(string)
	if !ok || name == "" {
		return "", fmt.Errorf("plugin.json is missing: %s", key)
	}
	return name, nil
}

// GetExecutableFromPluginJSON retrieves the executable from a plugin.json file in the provided directory.
// If the executable is not found in the root of the directory, it will look in a nested 'datasource' directory.
// If an executable value is found, it will call filepath.Base on the value to ensure that the executable is returned without any path information.
func GetExecutableFromPluginJSON(dir string) (string, error) {
	exe, err := GetStringValueFromJSON(path.Join(dir, "plugin.json"), "executable")
	if err != nil {
		// In app plugins, the nested plugin executable may be nested in a datasource directory
		exe, err2 := GetStringValueFromJSON(path.Join(dir, "datasource", "plugin.json"), "executable")
		if err2 != nil {
			return "", err
		}
		return filepath.Join("datasource", filepath.Base(exe)), nil
	}
	return filepath.Base(exe), err
}
