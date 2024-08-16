package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestWithContextualAttributesForOutgoingContext(t *testing.T) {
	tcs := []struct {
		name      string
		logParams []any
		expected  []string
	}{
		{
			name:      "empty log params",
			logParams: []any{},
			expected:  []string{},
		},
		{
			name:      "log params with odd number of elements",
			logParams: []any{"key1", "value1", "key2"},
			expected:  []string{},
		},
		{
			name:      "log params with empty key",
			logParams: []any{"", "value1"},
			expected:  []string{},
		},
		{
			name:      "log params with empty value",
			logParams: []any{"key1", ""},
			expected:  []string{},
		},
		{
			name:      "log params with valid key and value",
			logParams: []any{"key1", "value1"},
			expected:  []string{logParam("key1", "value1")},
		},
		{
			name:      "log params with multiple key value pairs",
			logParams: []any{"key1", "value1", "key2", "value2"},
			expected:  []string{logParam("key1", "value1"), logParam("key2", "value2")},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := WithContextualAttributesForOutgoingContext(context.Background(), tc.logParams)
			md, ok := metadata.FromOutgoingContext(ctx)
			if len(tc.expected) == 0 {
				require.False(t, ok)
				return
			}

			require.True(t, ok)
			got := md.Get(logParamsCtxMetadataKey)
			if len(got) != len(tc.expected) {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Fatalf("expected %v, got %v", tc.expected, got)
				}
			}
		})
	}
}

func TestContextualAttributesFromIncomingContext(t *testing.T) {
	tcs := []struct {
		name     string
		md       metadata.MD
		expected []any
	}{
		{
			name:     "empty metadata",
			md:       metadata.MD{},
			expected: nil,
		},
		{
			name:     "metadata without log params",
			md:       metadata.MD{"key1": []string{"value1"}},
			expected: nil,
		},
		{
			name:     "metadata with valid log params",
			md:       metadata.MD{logParamsCtxMetadataKey: []string{logParam("key1", "value1"), logParam("key2", "value2")}},
			expected: []any{"key1", "value1", "key2", "value2"},
		},
		{
			name:     "metadata with missing key",
			md:       metadata.MD{logParamsCtxMetadataKey: []string{logParam("", "value1"), logParam("key2", "value2")}},
			expected: []any{"key2", "value2"},
		},
		{
			name:     "metadata with missing value",
			md:       metadata.MD{logParamsCtxMetadataKey: []string{logParam("key1", ""), logParam("key2", "value2")}},
			expected: []any{"key2", "value2"},
		},
		{
			name:     "metadata with invalid key + value",
			md:       metadata.MD{logParamsCtxMetadataKey: []string{logParam("", ""), logParam("key2", "value2")}},
			expected: []any{"key2", "value2"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), tc.md)
			got := ContextualAttributesFromIncomingContext(ctx)
			require.Equal(t, tc.expected, got)
		})
	}
}
