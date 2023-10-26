package backend

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponse(t *testing.T) {
	for _, tc := range []struct {
		name            string
		err             error
		expStatus       Status
		expErrorMessage string
		expErrorSource  ErrorSource
	}{
		{
			name:            "generic error",
			err:             errors.New("other"),
			expStatus:       StatusUnknown,
			expErrorMessage: "other",
			expErrorSource:  ErrorSourcePlugin,
		},
		{
			name:            "downstream error",
			err:             DownstreamError(errors.New("bad gateway"), false),
			expStatus:       0,
			expErrorMessage: "bad gateway",
			expErrorSource:  ErrorSourceDownstream,
		},
		{
			name:            "plugin error",
			err:             PluginError(errors.New("internal error"), false),
			expStatus:       0,
			expErrorMessage: "internal error",
			expErrorSource:  ErrorSourcePlugin,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := Response(tc.err)
			require.Error(t, res.Error)
			require.Equal(t, tc.expStatus, res.Status)
			require.Equal(t, tc.expErrorMessage, res.Error.Error())
			require.Equal(t, tc.expErrorSource, res.ErrorSource)
		})
	}
}

func TestResponseWithOptions(t *testing.T) {
	unknown := NewError(errors.New("unknown"), ErrorSourcePlugin, StatusUnknown)
	badgateway := NewError(errors.New("bad gateway"), ErrorSourceDownstream, StatusBadGateway)

	for _, tc := range []struct {
		name            string
		err             Error
		expStatus       Status
		expErrorMessage string
		expErrorSource  ErrorSource
	}{
		{
			name:            "unknown error",
			err:             unknown,
			expStatus:       StatusUnknown,
			expErrorMessage: unknown.Error(),
			expErrorSource:  ErrorSourcePlugin,
		},
		{
			name:            "bad gateway",
			err:             badgateway,
			expStatus:       StatusBadGateway,
			expErrorMessage: badgateway.Error(),
			expErrorSource:  ErrorSourceDownstream,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := Response(tc.err)
			require.Error(t, res.Error)
			require.Equal(t, tc.expStatus, res.Status)
			require.Equal(t, tc.expErrorMessage, res.Error.Error())
			require.Equal(t, tc.expErrorSource, res.ErrorSource)
		})
	}
}
