package hash

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

var ErrCouldNotComputeDataSourceSettingsHash = errors.New("could not compute expectedHash for datasource settings")

// dsSettingHashTarget is a collection of fields from backend.DataSourceInstanceSettings which are used in combination
// to universally identify a datasource setting. Its primary purpose is to be used to generate a digest.
type dsSettingHashTarget struct {
	ID                      int64
	UID                     string
	Type                    string
	Name                    string
	URL                     string
	User                    string
	BasicAuthEnabled        bool
	BasicAuthUser           string
	JSONData                json.RawMessage
	DecryptedSecureJSONData map[string]string
}

func newDataSourceSettingHashTarget(settings backend.DataSourceInstanceSettings) dsSettingHashTarget {
	return dsSettingHashTarget{
		ID:                      settings.ID,
		UID:                     settings.UID,
		Type:                    settings.Type,
		Name:                    settings.Name,
		URL:                     settings.URL,
		User:                    settings.User,
		BasicAuthEnabled:        settings.BasicAuthEnabled,
		BasicAuthUser:           settings.BasicAuthUser,
		JSONData:                settings.JSONData,
		DecryptedSecureJSONData: settings.DecryptedSecureJSONData,
	}
}

// HashDataSourceSettings provides the expectedHash value of a backend.DataSourceInstanceSettings.
func HashDataSourceSettings(s backend.DataSourceInstanceSettings) (string, error) {
	hash, err := hashstructure.Hash(newDataSourceSettingHashTarget(s), hashstructure.FormatV2, nil)
	if err != nil {
		return "", ErrCouldNotComputeDataSourceSettingsHash
	}
	return strconv.FormatUint(hash, 10), nil
}
