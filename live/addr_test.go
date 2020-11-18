package live

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseChannelAddress_Valid(t *testing.T) {
	addr := ParseChannelAddress("aaa/bbb/ccc/ddd")
	require.True(t, addr.IsValid())

	ex := ChannelAddress{
		Scope:     "aaa",
		Namespace: "bbb",
		Path:      "ccc/ddd",
	}

	if diff := cmp.Diff(addr, ex); diff != "" {
		t.Fatalf("Result mismatch (-want +got):\n%s", diff)
	}
}

func TestParseChannelAddress_Invalid(t *testing.T) {
	addr := ParseChannelAddress("aaa/bbb")
	require.False(t, addr.IsValid())
}

func TestToWebSocketURL(t *testing.T) {
	testCases := []struct {
		desc   string
		url    string
		exp    string
		expErr string
	}{
		{
			desc: "Simple localhost",
			url:  "http://localhost:3000",
			exp:  "ws://localhost:3000/live/ws?format=protobuf",
		},
		{
			desc: "With subpath",
			url:  "http://host/with/subpath",
			exp:  "ws://host/with/subpath/live/ws?format=protobuf",
		},
		{
			desc:   "Invalid URL",
			url:    "xyz://asgasg:abc/with/subpath",
			expErr: `parse "xyz://asgasg:abc/with/subpath": invalid port ":abc" after host`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			conn := ConnectionInfo{
				URL: tc.url,
			}
			t.Log("Testing conn.ToWebSocketURL", "url", tc.url, "exp", tc.exp, "expErr", tc.expErr)
			ws, err := conn.ToWebSocketURL()
			if tc.expErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.exp, ws, tc.desc)
			} else {
				assert.EqualError(t, err, tc.expErr, tc.desc)
			}
		})
	}
}
