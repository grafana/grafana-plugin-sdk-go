package live

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestConnectionConversions(t *testing.T) {
	// Simple localhost
	conn := ConnectionInfo{
		URL: "http://localhost:3000",
	}
	ws, _ := conn.ToWebSocketURL()
	expect := "ws://localhost:3000/live/ws?format=protobuf"
	if diff := cmp.Diff(expect, ws); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}

	// Now with subpath
	conn.URL = "http://host/with/subpath"
	ws, _ = conn.ToWebSocketURL()
	expect = "ws://host/with/subpathlive/ws?format=protobuf"
	if diff := cmp.Diff(expect, ws); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}

	// Error parsing URL
	conn.URL = "xyz://asgasg:abc/with/subpath"
	_, err := conn.ToWebSocketURL()
	if err == nil {
		t.Fatalf("expected error parsing url")
	}
}
