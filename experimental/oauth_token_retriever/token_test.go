package oauthtokenretriever

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetExternalServiceToken(t *testing.T) {
	for _, test := range []struct {
		name   string
		userID string
	}{
		{"On Behalf Of", "1"},
		{"Service account", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				if test.userID != "" {
					assert.Contains(t, string(b), "assertion=")
					assert.Contains(t, string(b), "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Ajwt-bearer")
				} else {
					assert.NotContains(t, string(b), "assertion=")
					assert.Contains(t, string(b), "grant_type=client_credentials")
				}
				assert.Contains(t, string(b), "client_id=test_client_id")
				assert.Contains(t, string(b), "client_secret=test_client_secret")

				_, err = w.Write([]byte(`{"access_token":"test_token"}`))
				assert.NoError(t, err)
			}))
			httpCli := s.Client()

			ss, err := New(httpCli, s.URL, "test_client_id", "test_client_secret", testECDSAKey)
			assert.NoError(t, err)

			token, err := ss.GetExternalServiceToken(test.userID)
			assert.NoError(t, err)
			assert.Equal(t, "test_token", token)
		})
	}
}
