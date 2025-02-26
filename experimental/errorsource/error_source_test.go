package errorsource

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestResponse(t *testing.T) {
	for _, tc := range []struct {
		name            string
		err             error
		expStatus       backend.Status
		expErrorMessage string
		expErrorSource  backend.ErrorSource
	}{
		{
			name:            "generic error",
			err:             errors.New("other"),
			expErrorMessage: "other",
			expErrorSource:  "",
		},
		{
			name:            "downstream error",
			err:             DownstreamError(errors.New("bad gateway"), false),
			expErrorMessage: "bad gateway",
			expErrorSource:  backend.ErrorSourceDownstream,
		},
		{
			name:            "plugin error",
			err:             PluginError(errors.New("internal error"), false),
			expErrorMessage: "internal error",
			expErrorSource:  backend.ErrorSourcePlugin,
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
	unknown := New(errors.New("unknown"), backend.ErrorSourcePlugin, backend.StatusUnknown)
	badgateway := New(errors.New("bad gateway"), backend.ErrorSourceDownstream, backend.StatusBadGateway)

	for _, tc := range []struct {
		name            string
		err             Error
		expStatus       backend.Status
		expErrorMessage string
		expErrorSource  backend.ErrorSource
	}{
		{
			name:            "unknown error",
			err:             unknown,
			expStatus:       backend.StatusUnknown,
			expErrorMessage: unknown.Error(),
			expErrorSource:  backend.ErrorSourcePlugin,
		},
		{
			name:            "bad gateway",
			err:             badgateway,
			expStatus:       backend.StatusBadGateway,
			expErrorMessage: badgateway.Error(),
			expErrorSource:  backend.ErrorSourceDownstream,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := Response(tc.err)
			require.Error(t, res.Error)
			require.Equal(t, tc.expErrorMessage, res.Error.Error())
			require.Equal(t, tc.expErrorSource, res.ErrorSource)
		})
	}
}

func TestError(t *testing.T) {
	err := errors.New("boom")
	require.False(t, backend.IsDownstreamError(err))
	pErr := PluginError(err, true)
	require.False(t, backend.IsDownstreamError(pErr))
	dErr := DownstreamError(err, true)
	require.True(t, backend.IsDownstreamError(dErr))
}

func TestNilError(t *testing.T) {
	err := error(nil)
	pErr := PluginError(err, true)
	require.Nil(t, pErr)
	dErr := DownstreamError(err, true)
	require.Nil(t, dErr)
}
