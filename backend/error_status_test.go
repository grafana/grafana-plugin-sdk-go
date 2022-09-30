package backend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError_Error(t *testing.T) {
	tcs := []struct {
		name        string
		err         Error
		expectedErr Error
	}{
		{
			name:        "An empty error is considered an unknown error",
			err:         Error{},
			expectedErr: Error{status: ErrorStatusUnknown},
		},
		{
			name:        "An error with an invalid status is considered an unknown error",
			err:         Error{status: "invalidStatus"},
			expectedErr: Error{status: ErrorStatusUnknown},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedErr.Error(), tc.err.Error())
		})
	}
}
