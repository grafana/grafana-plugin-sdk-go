package pluginv3example_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginv3example"
	pluginv3 "github.com/grafana/grafana-plugin-sdk-go/genproto/grafana/plugin/v3"
)

// TestValidateAdmission covers the unary RPC the example overrides.
func TestValidateAdmission(t *testing.T) {
	p := &pluginv3example.Plugin{}

	resp, err := p.ValidateAdmission(context.Background(), pluginv3.ValidateAdmissionRequest_builder{
		Operation: pluginv3.Operation_OPERATION_CREATE.Enum(),
	}.Build())
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetAllowed() {
		t.Fatal("expected allowed = true")
	}
}

// TestUnimplementedRPCs documents the core of the "embed UnimplementedV3Server
// and override only what you serve" pattern: RPCs the example does not override
// fall through to the stub and report codes.Unimplemented rather than panicking
// or failing to compile.
func TestUnimplementedRPCs(t *testing.T) {
	p := &pluginv3example.Plugin{}
	ctx := context.Background()

	if _, err := p.MutateAdmission(ctx, pluginv3.MutateAdmissionRequest_builder{}.Build()); status.Code(err) != codes.Unimplemented {
		t.Fatalf("MutateAdmission: expected codes.Unimplemented, got %v", err)
	}

	if _, err := p.ConvertObjects(ctx, pluginv3.ConvertObjectsRequest_builder{}.Build()); status.Code(err) != codes.Unimplemented {
		t.Fatalf("ConvertObjects: expected codes.Unimplemented, got %v", err)
	}
}
