package v0alpha1

import (
	"errors"
	"fmt"
)

// Represents the state of some operation.
type Status struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`

	// Status is a textual representation of the state.
	Status string `json:"status"`
	// Message is a human-readable description of the state. It should not be parsed.
	Message string `json:"message"`
	// Code is a numeric code describing the state, such as an HTTP status code.
	Code int `json:"code"`
}

// Error produces a Go error from a status. It assumes the status represents failure.
func (s Status) Error() error {
	return errors.New(s.String())
}

// String implements stringer
func (s Status) String() string {
	return fmt.Sprintf("%s(%d): %s", s.Status, s.Code, s.Message)
}
