package backend

import (
	"fmt"
)

type DatasourceErrataError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	URL     string                 `json:"url"`
	Guide   string                 `json:"guide"`
	Args    map[string]interface{} `json:"args"`
}

func (r *DatasourceErrataError) Error() string {
	return fmt.Sprintf("%v", r.Message)
}
