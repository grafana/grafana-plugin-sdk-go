package harcapture

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestBuildSDKHAREntry_redactsSensitiveHeaders asserts that authentication-shaped headers are
// redacted before they reach the captured HAR entry -- this frame is returned to whoever enabled
// capture, so it must never carry the datasource's real credentials.
func TestBuildSDKHAREntry_redactsSensitiveHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer super-secret-token")
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	req.Header.Set("X-Api-Key", "sk-live-abc123")
	req.Header.Set("X-Custom", "keep-me")

	rec := httptest.NewRecorder()
	rec.Header().Set("Set-Cookie", "sid=super-secret-session")
	rec.Header().Set("X-Custom-Resp", "keep-me-too")
	rec.WriteHeader(http.StatusOK)
	resp := rec.Result()

	entry := buildSDKHAREntry(req, nil, false, resp, nil, time.Now(), time.Millisecond)

	find := func(pairs []sdkHARNameValue, name string) (string, bool) {
		for _, p := range pairs {
			if p.Name == name {
				return p.Value, true
			}
		}
		return "", false
	}

	for _, name := range []string{"Authorization", "Proxy-Authorization", "X-Api-Key"} {
		v, ok := find(entry.Request.Headers, name)
		if !ok {
			t.Fatalf("expected header %q to be captured (redacted), missing entirely", name)
		}
		if v != redactedValue {
			t.Errorf("header %q = %q, want redacted value %q", name, v, redactedValue)
		}
	}
	if v, ok := find(entry.Request.Headers, "X-Custom"); !ok || v != "keep-me" {
		t.Errorf("non-sensitive header X-Custom = %q (ok=%v), want unredacted %q", v, ok, "keep-me")
	}

	if v, ok := find(entry.Response.Headers, "Set-Cookie"); !ok || v != redactedValue {
		t.Errorf("response Set-Cookie header = %q (ok=%v), want redacted value %q", v, ok, redactedValue)
	}
	if v, ok := find(entry.Response.Headers, "X-Custom-Resp"); !ok || v != "keep-me-too" {
		t.Errorf("non-sensitive response header X-Custom-Resp = %q (ok=%v), want unredacted %q", v, ok, "keep-me-too")
	}
}

// TestBuildSDKHAREntry_redactsCookieValues asserts that parsed request/response cookie values are
// always redacted (name preserved), since a cookie's value is itself typically the credential.
func TestBuildSDKHAREntry_redactsCookieValues(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Secure/HttpOnly/SameSite are set only to satisfy gosec G124; they are response-cookie
	// attributes and are dropped when this is serialized into the outgoing Cookie header.
	req.AddCookie(&http.Cookie{Name: "session", Value: "super-secret-session-id", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})

	rec := httptest.NewRecorder()
	rec.Header().Set("Set-Cookie", "sid=another-secret-value")
	rec.WriteHeader(http.StatusOK)
	resp := rec.Result()

	entry := buildSDKHAREntry(req, nil, false, resp, nil, time.Now(), time.Millisecond)

	if len(entry.Request.Cookies) != 1 || entry.Request.Cookies[0].Name != "session" {
		t.Fatalf("request cookie not captured as expected: %+v", entry.Request.Cookies)
	}
	if entry.Request.Cookies[0].Value != redactedValue {
		t.Errorf("request cookie value = %q, want redacted value %q", entry.Request.Cookies[0].Value, redactedValue)
	}

	if len(entry.Response.Cookies) != 1 || entry.Response.Cookies[0].Name != "sid" {
		t.Fatalf("response cookie not captured as expected: %+v", entry.Response.Cookies)
	}
	if entry.Response.Cookies[0].Value != redactedValue {
		t.Errorf("response cookie value = %q, want redacted value %q", entry.Response.Cookies[0].Value, redactedValue)
	}
}

// TestBuildSDKHAREntry_redactsSensitiveQueryParams asserts that credential-shaped query string
// parameters (API keys, tokens, signed-URL signatures) are redacted, while ordinary params are left
// untouched.
func TestBuildSDKHAREntry_redactsSensitiveQueryParams(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://ds.example.com/query?api_key=sk-live-abc123&token=eyJhbGciOi&region=us-east-1", nil)
	if err != nil {
		t.Fatal(err)
	}

	entry := buildSDKHAREntry(req, nil, false, &http.Response{Header: http.Header{}}, nil, time.Now(), time.Millisecond)

	find := func(name string) (string, bool) {
		for _, p := range entry.Request.QueryString {
			if p.Name == name {
				return p.Value, true
			}
		}
		return "", false
	}

	for _, name := range []string{"api_key", "token"} {
		v, ok := find(name)
		if !ok {
			t.Fatalf("expected query param %q to be captured (redacted), missing entirely", name)
		}
		if v != redactedValue {
			t.Errorf("query param %q = %q, want redacted value %q", name, v, redactedValue)
		}
	}
	if v, ok := find("region"); !ok || v != "us-east-1" {
		t.Errorf("non-sensitive query param region = %q (ok=%v), want unredacted %q", v, ok, "us-east-1")
	}
}

// TestIsSensitiveHeaderName_caseInsensitive asserts the sensitive-header match ignores case, since
// headers can arrive in any casing (e.g. via a map built without http.Header's canonicalization).
func TestIsSensitiveHeaderName_caseInsensitive(t *testing.T) {
	for _, name := range []string{"authorization", "AUTHORIZATION", "Authorization", "AuthoRization"} {
		if !isSensitiveHeaderName(name) {
			t.Errorf("isSensitiveHeaderName(%q) = false, want true", name)
		}
	}
	if isSensitiveHeaderName("X-Custom") {
		t.Error("isSensitiveHeaderName(\"X-Custom\") = true, want false")
	}
}
