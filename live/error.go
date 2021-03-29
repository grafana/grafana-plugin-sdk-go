package live

import "fmt"

// StatusCodeError returned when Grafana Live returned an unexpected status code.
type StatusCodeError struct {
	Code int
}

// Error to implement error interface.
func (e StatusCodeError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.Code)
}
