package schemabuilder

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// admissionWatchlistSpec implements both Validate and Mutate: Validate
// rejects an empty severity, Mutate defaults it. The pairing exercises the
// dispatcher contract that mutation must not run validation (the server runs
// the validation phase after mutation).
type admissionWatchlistSpec struct {
	Title    string `json:"title"`
	Severity string `json:"severity,omitempty"`
}

func (s *admissionWatchlistSpec) Validate() error {
	if s.Title == "" {
		return errors.New("title is required")
	}
	if s.Severity == "" {
		return errors.New("severity is required")
	}
	return nil
}

func (s *admissionWatchlistSpec) Mutate() error {
	if s.Severity == "" {
		s.Severity = "info"
	}
	return nil
}

// admissionPlainSpec has neither Validate nor Mutate; admission should be a
// pass-through.
type admissionPlainSpec struct {
	Name string `json:"name"`
}

func newTestAdmissionHandler() backend.AdmissionHandler {
	return AdmissionHandler(
		AdmissionEntry{Kind: "Watchlist", SpecType: reflect.TypeOf(admissionWatchlistSpec{})},
		AdmissionEntry{Kind: "Plain", SpecType: reflect.TypeOf(admissionPlainSpec{})},
	)
}

func watchlistEnvelope(t *testing.T, spec map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"apiVersion": "example.grafana.app/v0alpha1",
		"kind":       "Watchlist",
		"metadata":   map[string]any{"name": "test"},
		"spec":       spec,
	})
	require.NoError(t, err)
	return raw
}

func TestAdmissionHandler_ValidateAdmission_Create(t *testing.T) {
	handler := newTestAdmissionHandler()

	tests := []struct {
		name        string
		spec        map[string]any
		wantAllowed bool
		wantMessage string
	}{
		{
			name:        "valid spec is allowed",
			spec:        map[string]any{"title": "mine", "severity": "warn"},
			wantAllowed: true,
		},
		{
			name:        "invalid spec is denied with the validation error",
			spec:        map[string]any{"severity": "warn"},
			wantAllowed: false,
			wantMessage: "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := handler.ValidateAdmission(context.Background(), &backend.AdmissionRequest{
				Operation:   backend.AdmissionRequestCreate,
				Kind:        backend.GroupVersionKind{Kind: "Watchlist"},
				ObjectBytes: watchlistEnvelope(t, tt.spec),
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantAllowed, resp.Allowed)
			if tt.wantMessage != "" {
				require.NotNil(t, resp.Result)
				require.Equal(t, tt.wantMessage, resp.Result.Message)
			}
		})
	}
}

func TestAdmissionHandler_UnknownKind(t *testing.T) {
	handler := newTestAdmissionHandler()
	req := &backend.AdmissionRequest{
		Operation:   backend.AdmissionRequestCreate,
		Kind:        backend.GroupVersionKind{Kind: "Mystery"},
		ObjectBytes: []byte(`{"spec":{}}`),
	}

	vResp, err := handler.ValidateAdmission(context.Background(), req)
	require.NoError(t, err)
	require.False(t, vResp.Allowed)
	require.NotNil(t, vResp.Result)
	require.Contains(t, vResp.Result.Message, `unknown kind "Mystery"`)

	mResp, err := handler.MutateAdmission(context.Background(), req)
	require.NoError(t, err)
	require.False(t, mResp.Allowed)
	require.NotNil(t, mResp.Result)
	require.Contains(t, mResp.Result.Message, `unknown kind "Mystery"`)
}

func TestAdmissionHandler_ValidateAdmission_Delete(t *testing.T) {
	handler := newTestAdmissionHandler()

	t.Run("old object spec is validated", func(t *testing.T) {
		// The old object is invalid (missing title): a denial proves the
		// dispatcher decoded OldObjectBytes and ran Validate against it.
		resp, err := handler.ValidateAdmission(context.Background(), &backend.AdmissionRequest{
			Operation:      backend.AdmissionRequestDelete,
			Kind:           backend.GroupVersionKind{Kind: "Watchlist"},
			OldObjectBytes: watchlistEnvelope(t, map[string]any{"severity": "warn"}),
		})
		require.NoError(t, err)
		require.False(t, resp.Allowed)
		require.NotNil(t, resp.Result)
		require.Equal(t, "title is required", resp.Result.Message)
	})

	t.Run("valid old object is allowed", func(t *testing.T) {
		resp, err := handler.ValidateAdmission(context.Background(), &backend.AdmissionRequest{
			Operation:      backend.AdmissionRequestDelete,
			Kind:           backend.GroupVersionKind{Kind: "Watchlist"},
			OldObjectBytes: watchlistEnvelope(t, map[string]any{"title": "mine", "severity": "warn"}),
		})
		require.NoError(t, err)
		require.True(t, resp.Allowed)
	})

	t.Run("no object at all is allowed", func(t *testing.T) {
		resp, err := handler.ValidateAdmission(context.Background(), &backend.AdmissionRequest{
			Operation: backend.AdmissionRequestDelete,
			Kind:      backend.GroupVersionKind{Kind: "Watchlist"},
		})
		require.NoError(t, err)
		require.True(t, resp.Allowed)
	})
}

func TestAdmissionHandler_MutateAdmission_PreservesUnknownFields(t *testing.T) {
	handler := newTestAdmissionHandler()

	in := []byte(`{
		"apiVersion": "example.grafana.app/v0alpha1",
		"kind": "Watchlist",
		"metadata": {"name": "test", "labels": {"a": "b"}},
		"spec": {"title": "mine"},
		"status": {"matchCount": 3},
		"foo": "bar"
	}`)

	resp, err := handler.MutateAdmission(context.Background(), &backend.AdmissionRequest{
		Operation:   backend.AdmissionRequestCreate,
		Kind:        backend.GroupVersionKind{Kind: "Watchlist"},
		ObjectBytes: in,
	})
	require.NoError(t, err)
	require.True(t, resp.Allowed)
	require.NotEmpty(t, resp.ObjectBytes)

	var got map[string]any
	require.NoError(t, json.Unmarshal(resp.ObjectBytes, &got))
	require.Equal(t, map[string]any{
		"apiVersion": "example.grafana.app/v0alpha1",
		"kind":       "Watchlist",
		"metadata":   map[string]any{"name": "test", "labels": map[string]any{"a": "b"}},
		"spec":       map[string]any{"title": "mine", "severity": "info"},
		"status":     map[string]any{"matchCount": float64(3)},
		"foo":        "bar",
	}, got)
}

func TestAdmissionHandler_MutateAdmission_DoesNotValidate(t *testing.T) {
	handler := newTestAdmissionHandler()

	// Empty severity fails Validate but Mutate defaults it; mutation must
	// succeed because the dispatcher leaves validation to the server's
	// validation phase.
	resp, err := handler.MutateAdmission(context.Background(), &backend.AdmissionRequest{
		Operation:   backend.AdmissionRequestCreate,
		Kind:        backend.GroupVersionKind{Kind: "Watchlist"},
		ObjectBytes: watchlistEnvelope(t, map[string]any{"title": "mine"}),
	})
	require.NoError(t, err)
	require.True(t, resp.Allowed)
	require.Nil(t, resp.Result)

	var got struct {
		Spec admissionWatchlistSpec `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(resp.ObjectBytes, &got))
	require.Equal(t, "info", got.Spec.Severity)
}

func TestAdmissionHandler_MutateAdmission_NoMutateMethod(t *testing.T) {
	handler := newTestAdmissionHandler()

	raw, err := json.Marshal(map[string]any{
		"kind": "Plain",
		"spec": map[string]any{"name": "x"},
	})
	require.NoError(t, err)

	resp, err := handler.MutateAdmission(context.Background(), &backend.AdmissionRequest{
		Operation:   backend.AdmissionRequestCreate,
		Kind:        backend.GroupVersionKind{Kind: "Plain"},
		ObjectBytes: raw,
	})
	require.NoError(t, err)
	require.True(t, resp.Allowed)
	require.Empty(t, resp.ObjectBytes)
}

func TestAdmissionHandler_MutateAdmission_EmptyObjectPassesThrough(t *testing.T) {
	handler := newTestAdmissionHandler()

	resp, err := handler.MutateAdmission(context.Background(), &backend.AdmissionRequest{
		Operation: backend.AdmissionRequestDelete,
		Kind:      backend.GroupVersionKind{Kind: "Watchlist"},
	})
	require.NoError(t, err)
	require.True(t, resp.Allowed)
	require.Empty(t, resp.ObjectBytes)
}
