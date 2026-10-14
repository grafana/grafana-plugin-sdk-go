package licensing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestLicenseFromIncomingGRPCContext(t *testing.T) {
	info := LicenseInfo{LicenseToken: "token", AppURL: "https://grafana.example.com", ValidationKeys: "keys"}
	encoded, err := json.Marshal(info)
	require.NoError(t, err)

	tests := []struct {
		name  string
		ctx   context.Context
		want  LicenseInfo
		found bool
	}{
		{name: "no metadata", ctx: context.Background()},
		{name: "nil metadata", ctx: metadata.NewIncomingContext(context.Background(), nil)},
		{name: "missing header", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("other", "value"))},
		{name: "empty header", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", ""))},
		{name: "malformed JSON", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", "{"))},
		{name: "wrong field type", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", `{"licenseToken":123}`))},
		{name: "valid metadata", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", string(encoded))), want: info, found: true},
		{name: "first header wins", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", string(encoded), "license", "{")), want: info, found: true},
		{name: "no fallback to later header", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("license", "", "license", string(encoded)))},
		{name: "outgoing metadata ignored", ctx: metadata.NewOutgoingContext(context.Background(), metadata.Pairs("license", string(encoded)))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := LicenseFromIncomingGRPCContext(tt.ctx)
			require.Equal(t, tt.found, found)
			require.Equal(t, tt.want, got)
		})
	}
}
