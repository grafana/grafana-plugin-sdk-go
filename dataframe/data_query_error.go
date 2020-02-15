package dataframe

import "encoding/json"

// DataQueryError
type DataQueryError struct {
	// Short message (typically shown in the header)
	Message string `json:"message,omitempty"`

	// longer error message, shown in the body
	Details string `json:"details,omitempty"`
}

// DataQueryErrorFromJSON creates aDataQueryError from a json string
func DataQueryErrorFromJSON(jsonStr string) (*DataQueryError, error) {
	var m DataQueryError
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
