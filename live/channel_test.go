package live

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestParseChannelAddress_Valid(t *testing.T) {
	addr := ParseChannel("aaa/bbb/ccc/ddd")
	require.True(t, addr.IsValid())

	ex := Channel{
		Scope:     "aaa",
		Namespace: "bbb",
		Path:      "ccc/ddd",
	}

	if diff := cmp.Diff(addr, ex); diff != "" {
		t.Fatalf("Result mismatch (-want +got):\n%s", diff)
	}
}

func TestParseChannelAddress_Invalid(t *testing.T) {
	addr := ParseChannel("aaa/bbb")
	require.False(t, addr.IsValid())
}
